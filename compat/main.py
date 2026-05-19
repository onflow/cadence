from __future__ import annotations

import logging
import os
import shlex
import shutil
import stat
import subprocess
from contextlib import contextmanager
from dataclasses import dataclass, field
from pathlib import Path
from typing import List, Optional, Collection, Dict

import concurrent.futures
import tempfile

import click as click
import coloredlogs
import yaml
from dacite import from_dict

logger = logging.getLogger(__name__)

SEPARATOR = "=" * 60
SUITE_PATH = Path("suite").resolve()
CLI_PATH = SUITE_PATH / "flow-cli" / "cmd" / "flow"

@contextmanager
def cwd(path):
    oldpwd = os.getcwd()
    os.chdir(path)
    try:
        yield
    finally:
        os.chdir(oldpwd)

def version_replacement(name: str, version: Optional[str]) -> str:
    if version:
        return f'{name}@{version}'
    else:
        commit_hash = Git.ls_remote('https://' + name, "HEAD")
        return f'{name}@{commit_hash}'

def path_replacement() -> str:
    return shlex.quote(Path(__file__).parent.parent.absolute().resolve().as_posix())

def build_cli(
    cadence_version: Optional[str],
    flowgo_version: Optional[str],
    flowemulator_version: Optional[str]
):
    logger.info("Building CLI ...")

    with cwd(SUITE_PATH):
        working_dir = SUITE_PATH / "flow-cli"
        Git.clone("https://github.com/onflow/flow-cli.git", "master", working_dir)
        with cwd(working_dir):
            Go.replace_dependencies(
                cadence_version=cadence_version,
                flowgo_version=flowgo_version,
                flowemulator_version=flowemulator_version
            )
            logger.info("Compiling CLI binary")
            subprocess.run(
                ["make", "binary"],
                check=True
            )

    logger.info("Built CLI")

@dataclass
class GoTest:
    path: str
    command: str

    def run(
        self,
        working_dir: Path,
        prepare: bool,
        cadence_version: Optional[str],
        flowgo_version: Optional[str],
        flowemulator_version: Optional[str]
    ) -> bool:

        with cwd(working_dir / self.path):
            if prepare:
                Go.replace_dependencies(cadence_version, flowgo_version, flowemulator_version)

            result = subprocess.run(self.command, shell=True)
            return result.returncode == 0

@dataclass
class CadenceTest:
    path: str
    command: str

    def run(
        self,
        working_dir: Path,
        prepare: bool,
        cadence_version: Optional[str],
        flowgo_version: Optional[str],
        flowemulator_version: Optional[str]
    ) -> bool:

        env = os.environ.copy()
        env["PATH"] = f"{shlex.quote(str(CLI_PATH))}:{env['PATH']}"

        with cwd(working_dir / self.path):
            result = subprocess.run(self.command, shell=True, env=env)
            return result.returncode == 0

@dataclass
class Description:
    description: str
    url: str
    branch: str
    go_tests: List[GoTest] = field(default_factory=list)
    cadence_tests: List[CadenceTest] = field(default_factory=list)

    @staticmethod
    def load(name: str) -> Description:
        path = SUITE_PATH / (name + ".yaml")
        with path.open(mode="r") as f:
            data = yaml.safe_load(f)
            return Description.from_dict(data)

    @classmethod
    def from_dict(cls, data: Dict):
        return from_dict(data_class=cls, data=data)

    def run(
        self,
        name: str,
        prepare: bool,
        go_test: bool,
        cadence_test: bool,
        cadence_version: Optional[str],
        flowgo_version: Optional[str],
        flowemulator_version: Optional[str]
    ) -> (bool):

        logger.info(f"Running tests for {name} ...")

        working_dir = SUITE_PATH / name

        if prepare:
            Git.clone(self.url, self.branch, working_dir)

        go_tests_succeeded = True
        if go_test:
            for test in self.go_tests:
                if not test.run(
                    working_dir,
                    prepare=prepare,
                    cadence_version=cadence_version,
                    flowgo_version=flowgo_version,
                    flowemulator_version=flowemulator_version
                ):
                    go_tests_succeeded = False

        cadence_tests_succeeded = True
        if cadence_test:
            for test in self.cadence_tests:
                if not test.run(
                    working_dir,
                    prepare=prepare,
                    cadence_version=cadence_version,
                    flowgo_version=flowgo_version,
                    flowemulator_version=flowemulator_version
                ):
                    cadence_tests_succeeded = False

        return go_tests_succeeded and cadence_tests_succeeded

class Go:

    @staticmethod
    def mod_replace(original: str, replacement: str):
        logger.info(f"Replacing {original} with {replacement} in go.mod")
        subprocess.run(
            ["go", "mod", "edit", "-replace", f'{original}={replacement}'],
            check=True
        )
        Go.mod_tidy()

    @staticmethod
    def mod_tidy():
        subprocess.run(
            ["go", "mod", "tidy"],
            check=True
        )

    @staticmethod
    def replace_dependencies(
        cadence_version: Optional[str],
        flowgo_version: Optional[str],
        flowemulator_version: Optional[str]
    ):
        logger.info("Editing dependencies")
        Go.mod_replace(
            "github.com/onflow/cadence",
            version_replacement("github.com/onflow/cadence", cadence_version) if cadence_version else path_replacement(),
        )
        Go.mod_replace(
            "github.com/onflow/flow-go",
            version_replacement("github.com/onflow/flow-go", flowgo_version)
        )
        Go.mod_replace(
            "github.com/onflow/flow-emulator",
            version_replacement("github.com/onflow/flow-emulator", flowemulator_version)
        )

class Git:

    @staticmethod
    def clone(url: str, branch: str, working_dir: Path):
        if working_dir.exists():
            Git._clean(working_dir)

        logger.info(f"Cloning {url} ({branch})")

        subprocess.run(
            ["git", "clone", "--depth", "1", "--branch", branch, url, working_dir],
            check=True
        )

    @staticmethod
    def _clean(working_dir: Path):
        for root, dirs, files in os.walk(working_dir):
            for dir in dirs:
                os.chmod(os.path.join(root, dir), stat.S_IRUSR | stat.S_IWUSR | stat.S_IXUSR)
            for file in files:
                os.chmod(os.path.join(root, file), stat.S_IRUSR | stat.S_IWUSR)
        shutil.rmtree(working_dir)

    @staticmethod
    def get_head_ref() -> str:
        completed_process = subprocess.run(
            ["git", "rev-parse", "--abbrev-ref", "HEAD"],
            capture_output=True,
            check=True
        )
        return completed_process.stdout.decode("utf-8").strip()

    @staticmethod
    def ls_remote(url: str, branch: str) -> str:
        completed_process = subprocess.run(
            ["git", "ls-remote", url, branch],
            capture_output=True,
            check=True
        )
        output = completed_process.stdout.decode("utf-8").strip()
        return output.split("\t")[0] if output else ""

@click.command()
@click.option(
    "--rerun",
    is_flag=True,
    default=False,
    help="Rerun without cloning and preparing the suites"
)
@click.option(
    "--go-test/--no-go-test",
    is_flag=True,
    default=True,
    help="Run the suite Go tests"
)
@click.option(
    "--cadence-test/--no-cadence-test",
    is_flag=True,
    default=True,
    help="Run the suite Cadence tests"
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
    "--flowemulator-version",
    default=None,
    help="version of flow-emulator for Go tests"
)
@click.option(
    "--parallel/--no-parallel",
    is_flag=True,
    default=True,
    help="Run suites in parallel"
)
@click.argument(
    "names",
    nargs=-1,
)
def main(
    rerun: bool,
    go_test: bool,
    cadence_test: bool,
    cadence_version: str,
    flowgo_version: str,
    flowemulator_version: str,
    parallel: bool,
    names: Collection[str]
):

    logger.info(
        f'Chosen versions: '
        + f'cadence@{cadence_version if cadence_version else "local version"}, '
        + f'flow-go@{flowgo_version if flowgo_version else "HEAD"}, '
        + f'flow-emulator@{flowemulator_version if flowemulator_version else "HEAD"}'
    )

    prepare = not rerun

    if cadence_test and prepare:
        build_cli(
            cadence_version=cadence_version,
            flowgo_version=flowgo_version,
            flowemulator_version=flowemulator_version
        )

    # Run for the current checkout

    current_success = run(
        parallel=parallel,
        prepare=prepare,
        go_test=go_test,
        cadence_test=cadence_test,
        cadence_version=cadence_version,
        flowgo_version=flowgo_version,
        flowemulator_version=flowemulator_version,
        names=names
    )

    if not current_success:
        exit(1)


def _run_single_suite(name, prepare, go_test, cadence_test,
                      cadence_version, flowgo_version, flowemulator_version):
    """Run a single suite in a worker process, capturing all output."""
    stdout_file = tempfile.TemporaryFile()
    stderr_file = tempfile.TemporaryFile()
    old_stdout_fd = os.dup(1)
    old_stderr_fd = os.dup(2)
    os.dup2(stdout_file.fileno(), 1)
    os.dup2(stderr_file.fileno(), 2)

    try:
        coloredlogs.install(level="INFO", fmt="%(asctime)s,%(msecs)03d %(levelname)s %(message)s")
        description = Description.load(name)
        succeeded = description.run(
            name,
            prepare=prepare,
            go_test=go_test,
            cadence_test=cadence_test,
            cadence_version=cadence_version,
            flowgo_version=flowgo_version,
            flowemulator_version=flowemulator_version
        )
    except Exception:
        succeeded = False
    finally:
        os.dup2(old_stdout_fd, 1)
        os.dup2(old_stderr_fd, 2)
        os.close(old_stdout_fd)
        os.close(old_stderr_fd)

    stdout_file.seek(0)
    stderr_file.seek(0)
    output = stdout_file.read().decode("utf-8", errors="replace") \
           + stderr_file.read().decode("utf-8", errors="replace")
    stdout_file.close()
    stderr_file.close()

    return name, succeeded, output


def run(
    parallel: bool,
    prepare: bool,
    go_test: bool,
    cadence_test: bool,
    cadence_version: str,
    flowgo_version: str,
    flowemulator_version: str,
    names: Collection[str]
) -> bool:

    if not names:
        names = [f.stem for f in SUITE_PATH.glob("*.yaml")]

    results = {}

    if parallel:
        with concurrent.futures.ProcessPoolExecutor() as executor:
            futures = {}
            for name in names:
                logger.info(f"Starting suite: {name}")
                futures[executor.submit(
                    _run_single_suite,
                    name, prepare, go_test, cadence_test,
                    cadence_version, flowgo_version, flowemulator_version
                )] = name
            for future in concurrent.futures.as_completed(futures):
                name = futures[future]
                try:
                    name, succeeded, output = future.result()
                    results[name] = (succeeded, output)
                except Exception as e:
                    results[name] = (False, str(e))
                succeeded, _ = results[name]
                logger.info(f"Finished suite: {name} — {'PASS' if succeeded else 'FAIL'}")
    else:
        for name in names:
            description = Description.load(name)
            succeeded = description.run(
                name,
                prepare=prepare,
                go_test=go_test,
                cadence_test=cadence_test,
                cadence_version=cadence_version,
                flowgo_version=flowgo_version,
                flowemulator_version=flowemulator_version
            )
            results[name] = (succeeded, "")

    # Report
    passed = sum(1 for s, _ in results.values() if s)
    total = len(results)
    print("\n" + SEPARATOR)
    print("RESULTS")
    print(SEPARATOR)
    for name in names:
        succeeded, _ = results.get(name, (False, ""))
        status = "PASS" if succeeded else "FAIL"
        print(f"  {name:40s} {status}")
    print(SEPARATOR)
    print(f"Overall: {'PASS' if passed == total else 'FAIL'} ({passed}/{total} passed)")
    print(SEPARATOR)

    # Dump output for failed suites
    for name in names:
        succeeded, output = results.get(name, (False, ""))
        if not succeeded and output:
            print(f"\n{SEPARATOR}")
            print(f"OUTPUT: {name}")
            print(SEPARATOR)
            print(output)

    return all(s for s, _ in results.values())


if __name__ == "__main__":
    coloredlogs.install(
        level="INFO",
        fmt="%(asctime)s,%(msecs)03d %(levelname)s %(message)s"
    )

    main()
