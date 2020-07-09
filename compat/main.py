from __future__ import annotations

import os
from dataclasses import dataclass, field
from typing import List, Optional, Collection
from pathlib import Path
import logging
import subprocess

import click as click
import yaml
from dacite import from_dict
import coloredlogs


SUITE_PATH = Path('suite').resolve()

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

    @staticmethod
    def check(path: Path, working_dir: Path) -> bool:
        logger.info(f'Checking {path}')
        result = subprocess.run([CHECKER_PATH, path], cwd=str(path.parent))
        if result.returncode != 0:
            logger.error(f'Checking failed: {path}')
            return False

        return True

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
            return from_dict(data_class=Description, data=data)

    def _clone(self, working_dir: Path):
        if working_dir.exists():
            raise Exception(f'{working_dir} exists')

        clone_repository(self.url, self.branch, working_dir)

    def run(self, name: str, clone: bool) -> bool:
        working_dir = SUITE_PATH / name

        if clone:
            self._clone(working_dir)

        succeeded = True

        for file in self.files:
            path = working_dir.joinpath(file.path)

            if clone:
                file.rewrite(path)

            if not File.check(path, working_dir):
                succeeded = False

        return succeeded


def clone_repository(url: str, branch: str, working_dir: Path):
    logger.info(f'Cloning {url} ({branch})')
    subprocess.run([
        'git', 'clone', '--single-branch', '--branch',
        branch, url, working_dir
    ])


def build_checker():
    logger.info(f'Building checker')
    subprocess.run(['go', 'build', '../runtime/cmd/check'])


@click.command()
@click.option('--rerun', is_flag=True, default=False, help='Rerun without cloning')
@click.argument('names', nargs=-1)
def main(rerun: bool, names: Collection[str]):
    build_checker()

    failed = False

    if not len(names):
        names = [description_path.stem for description_path in SUITE_PATH.glob('*.yaml')]

    for name in names:

        description = Description.load(name)

        if not description.run(name, clone=not rerun):
            failed = True

    if failed:
        exit(1)


if __name__ == "__main__":

    logger = logging.getLogger(__name__)

    coloredlogs.install(
        level='INFO',
        fmt='%(asctime)s,%(msecs)03d %(levelname)s %(message)s'
    )

    main()
