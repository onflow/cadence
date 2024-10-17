from __future__ import annotations

import dataclasses
import html
import json
import logging
import os
import re
import shlex
import shutil
import stat
import subprocess
import sys
import textwrap
from collections import defaultdict
from contextlib import contextmanager
from dataclasses import dataclass, field
from enum import Enum, unique
from pathlib import Path
from typing import List, Optional, Collection, Dict, IO, Any
from typing_extensions import Protocol, Literal

import click as click
import coloredlogs
import tabulate
import yaml
from click.utils import LazyFile
from dacite import from_dict

SUITE_PATH = Path("suite").resolve()
CMD_PATH = Path("../runtime/cmd").resolve()
PARSER_PATH = Path("parse").resolve()
CHECKER_PATH = Path("check").resolve()

ansi_escape_pattern = re.compile(r'\x1b[^m]*m')


class Openable(Protocol):
    def open(self) -> IO:
        pass


@contextmanager
def cwd(path):
    oldpwd = os.getcwd()
    os.chdir(path)
    try:
        yield
    finally:
        os.chdir(oldpwd)


@dataclass
class File:
    path: str
    prepare: Optional[str]
    member_account_access: List[str] = field(default_factory=list)

    def rewrite(self, path: Path):
        if not isinstance(self.prepare, str):
            return

        logger.info(f"Preparing {path}")

        source: str
        with path.open(mode="r") as f:
            source = f.read()

        def replace(pattern, replacement):
            nonlocal source
            source = re.sub(pattern, replacement, source)

        variables = {"replace": replace}
        with cwd(str(path.parent.absolute())):
            exec(self.prepare, variables)

        with path.open(mode="w") as f:
            f.write(source)

    @staticmethod
    def _run(
            tool_path: Path,
            verb: str,
            path: Path,
            use_json: bool,
            bench: bool,
            extra_args: List[str] = None
    ) -> (bool, Optional):
        logger.info(f"{verb} {path}")
        json_args = ["-json"] if use_json else []
        bench_args = ["-bench"] if bench else []
        args = json_args + bench_args + (extra_args or [])

        completed_process = subprocess.run(
            [tool_path, *args, path],
            cwd=str(path.parent),
            capture_output=use_json
        )
        result = None
        if use_json:
            result = json.loads(completed_process.stdout)

        if completed_process.returncode != 0:
            logger.error(f"{verb} failed: {path}")
            return False, result

        return True, result


@dataclass
class GoTest:
    path: str
    command: str

    def run(self, working_dir: Path, prepare: bool, cadence_version: Optional[str], flowgo_version: Optional[str]) -> bool:
        if cadence_version:
            cadence_replacement = f'github.com/onflow/cadence@{cadence_version}'
        else:
            cadence_replacement = Path.cwd().parent.absolute().resolve().as_posix()

        if flowgo_version:
            flowgo_replacement = f'github.com/onflow/flow-go@{flowgo_version}'
        else:
            # default flow-go version is 0.37.16 the one corresponding to cadence v1.0.0
            flowgo_replacement = f'github.com/onflow/flow-go@v0.37.16'

        with cwd(working_dir / self.path):
            if prepare:
                logger.info("Editing dependencies")
                subprocess.run([
                    "go", "mod", "edit", "-replace", f'github.com/onflow/flow-go={flowgo_replacement}',
                ])
                subprocess.run([
                    "go", "mod", "edit", "-replace", f'github.com/onflow/cadence={cadence_replacement}',
                ])
                logger.info("Downloading dependencies")
                subprocess.run([
                    "go", "get", "-t", ".",
                ])

            result = subprocess.run(shlex.split(self.command))
            return result.returncode == 0


@dataclass
class BenchmarkResult:
    iterations: int
    time: int


@dataclass
class ParseResult:
    error: Optional[Any] = field(default=None)
    bench: Optional[BenchmarkResult] = field(default=None)

    @classmethod
    def from_dict(cls, data: Dict):
        return from_dict(data_class=cls, data=data)


@dataclass
class CheckResult:
    error: Optional[Any] = field(default=None)
    bench: Optional[BenchmarkResult] = field(default=None)

    @classmethod
    def from_dict(cls, data: Dict):
        return from_dict(data_class=cls, data=data)


@dataclass
class Result:
    path: str
    parse_result: ParseResult
    check_result: Optional[CheckResult] = field(default=None)


def load_index(path: Path) -> List[str]:
    logger.info(f"Loading suite index from {path} ...")
    with path.open(mode="r") as f:
        return yaml.safe_load(f)

@dataclass
class Description:
    description: str
    url: str
    branch: str
    files: List[File] = field(default_factory=list)
    go_tests: List[GoTest] = field(default_factory=list)

    @staticmethod
    def load(name: str) -> Description:
        path = SUITE_PATH / (name + ".yaml")
        with path.open(mode="r") as f:
            data = yaml.safe_load(f)
            return Description.from_dict(data)

    @classmethod
    def from_dict(cls, data: Dict):
        return from_dict(data_class=cls, data=data)

    def _clone(self, working_dir: Path):
        if working_dir.exists():
            for root, dirs, files in os.walk(working_dir):  
                for dir in dirs:
                    os.chmod(os.path.join(root, dir), stat.S_IRUSR | stat.S_IWUSR)
                for file in files:
                    os.chmod(os.path.join(root, file), stat.S_IRUSR | stat.S_IWUSR)
            shutil.rmtree(working_dir)

        logger.info(f"Cloning {self.url} ({self.branch})")

        Git.clone(self.url, self.branch, working_dir)

    def run(
        self,
        name: str,
        prepare: bool,
        use_json: bool,
        bench: bool,
        check: bool,
        go_test: bool,
        cadence_version: Optional[str],
        flowgo_version: Optional[str],
    ) -> (bool, List[Result]):

        working_dir = SUITE_PATH / name

        if prepare:
            self._clone(working_dir)

        results: List[Result] = []
        go_tests_succeeded = True
        if go_test:
            for test in self.go_tests:
                if not test.run(working_dir, prepare=prepare, cadence_version=cadence_version, flowgo_version=flowgo_version):
                    go_tests_succeeded = False

        succeeded = go_tests_succeeded

        return succeeded, results

class Git:

    @staticmethod
    def clone(url: str, branch: str, working_dir: Path):
        subprocess.run([
            "git", "clone", "--depth", "1", "--branch",
            branch, url, working_dir
        ])

    @staticmethod
    def get_head_ref() -> str:
        completed_process = subprocess.run(
            ["git", "rev-parse", "--abbrev-ref", "HEAD"],
            capture_output=True,
        )
        if completed_process.returncode != 0:
            raise Exception('failed to get current Git ref')
        return completed_process.stdout.decode("utf-8").strip()

    @staticmethod
    def checkout(ref: str):
        completed_process = subprocess.run(["git", "checkout", ref])
        if completed_process.returncode != 0:
            raise Exception(f'failed to checkout ref {ref}')


def build(name):
    logger.info(f"Building {name}")
    subprocess.run(["go", "build", CMD_PATH / name])


class EnhancedJSONEncoder(json.JSONEncoder):
    def default(self, o):
        if dataclasses.is_dataclass(o):
            return dataclasses.asdict(o)
        if isinstance(o, Enum):
            return o.value
        return super().default(o)


def indent(text: str) -> str:
    return textwrap.indent(text, "    ")


def format_markdown_details(details: str) -> str:
    stripped = ansi_escape_pattern.sub('', details)
    escaped = html.escape(stripped).replace("\n", "<br />")
    return f"<details><summary>Details</summary><pre>{escaped}</pre></details>"


@dataclass
class Comparisons:
    parse_error_comparisons: Dict[str, ErrorComparison]
    parse_bench_comparisons: Optional[Dict[str, BenchComparison]]
    check_error_comparisons: Dict[str, ErrorComparison]
    check_bench_comparisons: Optional[Dict[str, BenchComparison]]

    @classmethod
    def from_results(cls,
                     other_results: List[Result],
                     current_results: List[Result],
                     bench: bool,
                     delta_threshold: float
                     ) -> Comparisons:

        parse_error_comparisons: Dict[str, ErrorComparison] = {}
        check_error_comparisons: Dict[str, ErrorComparison] = {}

        parse_bench_comparisons: Optional[Dict[str, BenchComparison]] = None
        check_bench_comparisons: Optional[Dict[str, BenchComparison]] = None
        if bench:
            parse_bench_comparisons: Dict[str, BenchComparison] = {}
            check_bench_comparisons: Dict[str, BenchComparison] = {}

        other_results.sort(key=lambda result: result.path)
        current_results.sort(key=lambda result: result.path)

        for other_result, current_result in zip(other_results, current_results):
            path = other_result.path
            assert current_result.path == path

            parse_error_comparisons[path] = ErrorComparison(
                other_result.parse_result.error,
                current_result.parse_result.error
            )
            check_error_comparisons[path] = ErrorComparison(
                other_result.check_result.error,
                current_result.check_result.error
            )

            if bench:
                if other_result.parse_result.bench and current_result.parse_result.bench:
                    parse_bench_comparisons[path] = BenchComparison(
                        other_result.parse_result.bench,
                        current_result.parse_result.bench,
                        delta_threshold=delta_threshold
                    )

                if other_result.check_result.bench and current_result.check_result.bench:
                    check_bench_comparisons[path] = BenchComparison(
                        other_result.check_result.bench,
                        current_result.check_result.bench,
                        delta_threshold=delta_threshold
                    )

        return Comparisons(
            parse_error_comparisons,
            parse_bench_comparisons,
            check_error_comparisons,
            check_bench_comparisons
        )

    def write(self, output: IO, format: Format):

        result: Optional[str] = None
        if format in ("pretty", "markdown"):

            output.write("\n## Parser Errors\n\n")
            self._write_error_comparisons(self.parse_error_comparisons, output, format)
            if self.parse_bench_comparisons:
                output.write("\n## Parser Benchmarks\n\n")
                self._write_bench_comparisons(self.parse_bench_comparisons, output, format)

            output.write("\n## Checker Errors\n\n")
            self._write_error_comparisons(self.check_error_comparisons, output, format)
            if self.check_bench_comparisons:
                output.write("\n## Checker Benchmarks\n\n")
                self._write_bench_comparisons(self.check_bench_comparisons, output, format)

        if format == "json":
            result = json_serialize(self)

        if result:
            output.write(result)

    @staticmethod
    def _write_error_comparisons(comparisons: Dict[str, ErrorComparison], output: IO, format: Format):

        def write_table(data, headers):
            output.write(tabulate.tabulate(data, headers, tablefmt="pipe"))
            output.write("\n\n")

        groups = defaultdict(list)

        for path, comparison in comparisons.items():
            relative_path = Path(path).relative_to(SUITE_PATH)
            groups[comparison.category].append((relative_path, comparison))

        for category in (
                ErrorCategory.REGRESSION,
                ErrorCategory.STILL_FAIL,
                ErrorCategory.CHANGE,
                ErrorCategory.IMPROVEMENT,
                ErrorCategory.STILL_SUCCESS,
        ):

            category_comparisons = groups.get(category, [])

            if not len(category_comparisons):
                continue

            if format == "pretty":

                if category == ErrorCategory.REGRESSION:
                    output.write("😭 Regressions\n")
                    output.write("-------------\n")

                    for path, comparison in category_comparisons:
                        output.write(f"- {path}:\n")
                        output.write(indent(comparison.current))
                        output.write("\n\n")

                elif category == ErrorCategory.STILL_FAIL:
                    output.write("😢 Still failing\n")
                    output.write("---------------\n")

                    for path, comparison in category_comparisons:
                        output.write(f"- {path}:\n")
                        output.write(indent(comparison.current))
                        output.write("\n\n")

                elif category == ErrorCategory.CHANGE:
                    output.write("😕 Changed\n")
                    output.write("---------\n")

                    for path, comparison in category_comparisons:
                        output.write(f"- {path}:\n")
                        output.write("  Before:\n")
                        output.write(indent(comparison.other))
                        output.write("\n")
                        output.write("  Now:\n")
                        output.write(indent(comparison.current))
                        output.write("\n\n")

                elif category == ErrorCategory.IMPROVEMENT:
                    output.write("🎉 Improvements\n")
                    output.write("--------------\n")

                    for path, comparison in category_comparisons:
                        output.write(f"- {path}\n")

                elif category == ErrorCategory.STILL_SUCCESS:
                    output.write("🙂 Still succeeding\n")
                    output.write("------------------\n")

                    for path, comparison in category_comparisons:
                        output.write(f"- {path}\n")

                output.write("\n")

            elif format == "markdown":

                if category == ErrorCategory.REGRESSION:
                    data = []
                    headers = ["😭 Regressions", "Details"]
                    for path, comparison in category_comparisons:
                        data.append([path, format_markdown_details(comparison.current)])

                    write_table(data, headers)

                elif category == ErrorCategory.STILL_FAIL:
                    data = []
                    headers = ["😢 Still failing", "Details"]
                    for path, comparison in category_comparisons:
                        data.append([path, format_markdown_details(comparison.current)])

                    write_table(data, headers)

                elif category == ErrorCategory.CHANGE:
                    data = []
                    headers = ["😕 Changed", "Before", "Now"]
                    for path, comparison in category_comparisons:
                        data.append([
                            path,
                            format_markdown_details(comparison.other),
                            format_markdown_details(comparison.current)
                        ])

                    write_table(data, headers)

                elif category == ErrorCategory.IMPROVEMENT:
                    data = []
                    headers = ["🎉 Improvements"]
                    for path, comparison in category_comparisons:
                        data.append([path])

                    write_table(data, headers)

                elif category == ErrorCategory.STILL_SUCCESS:
                    data = []
                    headers = ["🙂 Still succeeding"]
                    for path, comparison in category_comparisons:
                        data.append([path])

                    write_table(data, headers)

    @classmethod
    def _write_bench_comparisons(cls, comparisons: Dict[str, BenchComparison], output: IO, format: Format):

        def write_table(data, headers):
            output.write(tabulate.tabulate(data, headers, tablefmt="pipe"))
            output.write("\n\n")

        groups = defaultdict(list)

        for path, comparison in comparisons.items():
            relative_path = Path(path).relative_to(SUITE_PATH)
            groups[comparison.category].append((relative_path, comparison))

        for category in (
                BenchCategory.REGRESSION,
                BenchCategory.IMPROVEMENT,
                BenchCategory.SAME,
        ):

            category_comparisons = groups.get(category, [])

            if not len(category_comparisons):
                continue

            title = ""
            if category == BenchCategory.REGRESSION:
                title = "😭 Regressions"
            elif category == BenchCategory.IMPROVEMENT:
                title = "🎉 Improvements"
            elif category == BenchCategory.SAME:
                title = "🙂 Same"

            data = []
            headers = [title, "OLD", "NEW", "DELTA", "RATIO"]
            for path, comparison in category_comparisons:
                data.append([
                    path,
                    cls._time_markdown(comparison.other.time),
                    cls._time_markdown(comparison.current.time),
                    cls._delta_markdown(comparison.delta),
                    cls._ratio_markdown(comparison.ratio)
                ])

            write_table(data, headers)

    @staticmethod
    def _time_markdown(time: int) -> str:
        return f'{time / 1000000:.2f}'

    @staticmethod
    def _delta_markdown(delta: float) -> str:
        result = f'{delta:.2f}%'
        if result[0] != '-':
            result = '+' + result
        return result

    @staticmethod
    def _ratio_markdown(ratio: float) -> str:
        return f'**{ratio:.2f}x**'


@unique
class ErrorCategory(Enum):
    REGRESSION = "Regression"
    IMPROVEMENT = "Improvement"
    CHANGE = "Change"
    STILL_SUCCESS = "Still succeeding"
    STILL_FAIL = "Still failing"


@dataclass
class ErrorComparison:
    other: Optional[str]
    current: Optional[str]
    category: ErrorCategory = field(init=False)

    def __post_init__(self):
        self.category = self._category()

    def _category(self) -> ErrorCategory:
        if self.other != self.current:
            return ErrorCategory.CHANGE
        elif not self.other and self.current:
            return ErrorCategory.REGRESSION
        elif self.other and not self.current:
            return ErrorCategory.IMPROVEMENT
        elif self.other:
            return ErrorCategory.STILL_FAIL
        else:
            return ErrorCategory.STILL_SUCCESS


@unique
class BenchCategory(Enum):
    REGRESSION = "Regression"
    IMPROVEMENT = "Improvement"
    SAME = "Same"


@dataclass
class BenchComparison:
    other: BenchmarkResult
    current: BenchmarkResult
    delta_threshold: float
    ratio: float = field(init=False)
    delta: float = field(init=False)
    category: BenchCategory = field(init=False)

    def __post_init__(self):
        self.ratio = self.other.time / self.current.time
        self.delta = -(((self.current.time * 100) / self.other.time) - 100)
        self.category = self._category()

    def _category(self) -> BenchCategory:
        if abs(self.delta) > self.delta_threshold:
            if self.delta > 0:
                return BenchCategory.IMPROVEMENT
            if self.delta < 0:
                return BenchCategory.REGRESSION

        return BenchCategory.SAME


Format = Literal["pretty", "json", "markdown"]


def json_serialize(data) -> str:
    return json.dumps(data, indent=4, cls=EnhancedJSONEncoder)

@click.command()
@click.option(
    "--rerun",
    is_flag=True,
    default=False,
    help="Rerun without cloning and preparing the suites"
)
@click.option(
    "--format",
    type=click.Choice(["pretty", "json", "markdown"], case_sensitive=False),
    default="pretty",
    help="output format",
)
@click.option(
    "--bench/--no-bench",
    is_flag=True,
    default=False,
    help="Run benchmarks"
)
@click.option(
    "--check/--no-check",
    is_flag=True,
    default=True,
    help="Parse and check the suite files"
)
@click.option(
    "--go-test/--no-go-test",
    is_flag=True,
    default=True,
    help="Run the suite Go tests"
)
@click.option(
    "--cadence-version",
    default=None,
    help="version of Cadence for Go tests"
)
@click.option(
    "--flowgo-version",
    default=None,
    help="version of flow-go for Go tests"
)
@click.option(
    "--output",
    default=None,
    type=click.File("w"),
    help="Write output to given path. Standard output by default"
)
@click.option(
    "--compare-ref",
    "other_ref",
    help="Compare with another Git ref (e.g. commit or branch)"
)
@click.option(
    "--delta-threshold",
    default=4,
    type=float,
    help="Delta threshold to consider a benchmark result a change"
)
@click.argument(
    "names",
    nargs=-1,
)
def main(
        rerun: bool,
        format: Format,
        bench: bool,
        check: bool,
        go_test: bool,
        cadence_version: str,
        flowgo_version: str,
        output: Optional[LazyFile],
        other_ref: Optional[str],
        delta_threshold: float,
        names: Collection[str]
):
    if other_ref is None and format not in ("pretty", "json"):
        raise Exception(f"unsupported format: {format}")

    prepare = not rerun

    output: IO = output.open() if output else sys.stdout

    # Comparison of different runs is only possible when using JSON

    use_json_for_run = format != "pretty" or (other_ref is not None)

    # Run for the current checkout

    current_success, current_results = run(
        prepare=prepare,
        use_json=use_json_for_run,
        bench=bench,
        check=check,
        go_test=go_test,
        cadence_version=cadence_version,
        flowgo_version=flowgo_version,
        names=names
    )

    # Run for the other checkout, if any

    if other_ref:
        current_ref = Git.get_head_ref()
        try:
            Git.checkout(other_ref)

            _, other_results = run(
                # suite repositories were already cloned in the previous run
                prepare=False,
                use_json=use_json_for_run,
                bench=bench,
                check=check,
                go_test=go_test,
                names=names
            )

            comparisons = Comparisons.from_results(
                other_results,
                current_results,
                bench=bench,
                delta_threshold=delta_threshold,
            )

            comparisons.write(output, format)

        finally:
            Git.checkout(current_ref)

    else:

        if format == "json":
            results_json = json_serialize(current_results)
            output.write(results_json)

    output.close()

    if not current_success:
        exit(1)


def run(
        prepare: bool,
        use_json: bool,
        bench: bool,
        check: bool,
        go_test: bool,
        cadence_version: str,
        flowgo_version: str,
        names: Collection[str]
) -> (bool, List[Result]):

    all_succeeded = True

    logger.info(f'Chosen versions: cadence@{ cadence_version if cadence_version else "local version" }, flow-go@{flowgo_version if flowgo_version else "v0.37.16"}')

    if not names:
        names = load_index(SUITE_PATH / "index.yaml")

    all_results: List[Result] = []

    for name in names:

        description = Description.load(name)

        run_succeeded, results = description.run(
            name,
            prepare=prepare,
            use_json=use_json,
            bench=bench,
            check=check,
            go_test=go_test,
            cadence_version=cadence_version,
            flowgo_version=flowgo_version,
        )

        if not run_succeeded:
            all_succeeded = False

        all_results.extend(results)

    return all_succeeded, all_results


if __name__ == "__main__":
    logger = logging.getLogger(__name__)

    coloredlogs.install(
        level="INFO",
        fmt="%(asctime)s,%(msecs)03d %(levelname)s %(message)s"
    )

    main()
