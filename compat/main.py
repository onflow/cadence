from __future__ import annotations

import json
import os
import dataclasses
from dataclasses import dataclass, field
from typing import List, Optional, Collection, Any, Dict
from pathlib import Path
import logging
import subprocess

import click as click
import yaml
from click.utils import LazyFile
from dacite import from_dict
import coloredlogs

SUITE_PATH = Path('suite').resolve()

PARSER_PATH = Path('parse').resolve()
CHECKER_PATH = Path('check').resolve()


@dataclass
class File:
    path: str
    prepare: Optional[str]

    def rewrite(self, path: Path):
        if not isinstance(self.prepare, str):
            return

        logger.info(f'Preparing {path}')

        source: str
        with path.open(mode='r') as f:
            source = f.read()

        variables = {'source': source}
        os.chdir(str(path.parent.absolute()))
        exec(self.prepare, variables)
        source = variables['source']

        with path.open(mode='w') as f:
            f.write(source)

    @classmethod
    def parse(cls, path: Path, use_json: bool, bench: bool) -> (bool, Optional):
        return cls._run(PARSER_PATH, "Parsing", path, use_json, bench)

    @classmethod
    def check(cls, path: Path, use_json: bool, bench: bool) -> (bool, Optional):
        return cls._run(CHECKER_PATH, "Checking", path, use_json, bench)

    @staticmethod
    def _run(tool_path: Path, verb: str, path: Path, use_json: bool, bench: bool) -> (bool, Optional):
        logger.info(f'{verb} {path}')
        json_args = ['-json'] if use_json else []
        bench_args = ['-bench'] if bench else []
        args = json_args + bench_args
        completed_process = subprocess.run(
            [tool_path, *args, path],
            cwd=str(path.parent),
            capture_output=use_json
        )
        result = None
        if use_json:
            result = json.loads(completed_process.stdout)

        if completed_process.returncode != 0:
            logger.error(f'{verb} failed: {path}')
            return False, result

        return True, result


@dataclass
class BenchmarkResult:
    iterations: int
    time: int


@dataclass
class ParseResult:
    error: Optional[str] = field(default=None)
    bench: Optional[BenchmarkResult] = field(default=None)

    @classmethod
    def from_dict(cls, data: Dict):
        return from_dict(data_class=cls, data=data)


@dataclass
class CheckResult:
    error: Optional[str] = field(default=None)
    bench: Optional[BenchmarkResult] = field(default=None)

    @classmethod
    def from_dict(cls, data: Dict):
        return from_dict(data_class=cls, data=data)


@dataclass
class Result:
    path: str
    parse_result: ParseResult
    check_result: Optional[CheckResult] = field(default=None)


@dataclass
class Description:
    description: str
    url: str
    branch: str
    files: List[File]

    @staticmethod
    def load(name: str) -> Description:
        path = SUITE_PATH / (name + '.yaml')
        with path.open(mode='r') as f:
            data = yaml.safe_load(f)
            return Description.from_dict(data)

    @classmethod
    def from_dict(cls, data: Dict):
        return from_dict(data_class=cls, data=data)

    def _clone(self, working_dir: Path):
        if working_dir.exists():
            raise Exception(f'{working_dir} exists')

        clone_repository(self.url, self.branch, working_dir)

    def run(self, name: str, clone: bool, use_json: bool, bench: bool) -> (bool, List):
        working_dir = SUITE_PATH / name

        if clone:
            self._clone(working_dir)

        run_succeeded = True

        results: List[Result] = []

        for file in self.files:
            path = working_dir.joinpath(file.path)

            if clone:
                file.rewrite(path)

            parse_succeeded, parse_results = \
                File.parse(path, use_json=use_json, bench=bench)
            result = Result(
                path=str(path),
                parse_result=ParseResult.from_dict(parse_results[0]) if parse_results else None
            )
            if not parse_succeeded:
                run_succeeded = False
                if use_json:
                    results.append(result)
                continue

            check_succeeded, check_results = \
                File.check(path, use_json=use_json, bench=bench)
            if check_results:
                result.check_result = CheckResult.from_dict(check_results[0])
            if use_json:
                results.append(result)
            if not check_succeeded:
                run_succeeded = False
                continue

        return run_succeeded, results


def clone_repository(url: str, branch: str, working_dir: Path):
    logger.info(f'Cloning {url} ({branch})')
    subprocess.run([
        'git', 'clone', '--single-branch', '--branch',
        branch, url, working_dir
    ])


def build(name):
    logger.info(f'Building {name}')
    subprocess.run(['go', 'build', Path('../runtime/cmd') / name])


class EnhancedJSONEncoder(json.JSONEncoder):
    def default(self, o):
        if dataclasses.is_dataclass(o):
            return dataclasses.asdict(o)
        return super().default(o)


@click.command()
@click.option('--rerun', is_flag=True, default=False, help='Rerun without cloning')
@click.option('--json', 'use_json', is_flag=True, default=False, help='JSON')
@click.option('--bench', is_flag=True, default=False, help='Run benchmarks')
@click.option('--output', default=None, type=click.File('w'))
@click.argument('names', nargs=-1)
def main(rerun: bool, use_json: bool, bench: bool, output: LazyFile, names: Collection[str]):
    clone = not rerun

    build('parse')
    build('check')

    all_succeeded = True

    if not len(names):
        names = [description_path.stem for description_path in SUITE_PATH.glob('*.yaml')]

    all_results: List[Result] = []

    for name in names:

        description = Description.load(name)

        run_succeeded, results = description.run(
            name,
            clone=clone,
            use_json=use_json,
            bench=bench
        )

        if not run_succeeded:
            all_succeeded = False

        all_results.extend(results)

    if use_json:
        results_json = json.dumps(all_results, indent=4, cls=EnhancedJSONEncoder)
        if output:
            with output.open() as f:
                f.write(results_json)
        else:
            print(results_json)

    if not all_succeeded:
        exit(1)


if __name__ == "__main__":
    logger = logging.getLogger(__name__)

    coloredlogs.install(
        level='INFO',
        fmt='%(asctime)s,%(msecs)03d %(levelname)s %(message)s'
    )

    main()
