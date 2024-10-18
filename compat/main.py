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

import click as click
import coloredlogs
import yaml
from dacite import from_dict

SUITE_PATH = Path("suite").resolve()

@contextmanager
def cwd(path):
    oldpwd = os.getcwd()
    os.chdir(path)
    try:
        yield
    finally:
        os.chdir(oldpwd)

@dataclass
class GoTest:
    path: str
    command: str

    def run(self, working_dir: Path, prepare: bool, cadence_version: Optional[str], flowgo_version: Optional[str]) -> bool:
        if cadence_version:
            cadence_replacement = f'github.com/onflow/cadence@{cadence_version}'
        else:
            # default: point to local cadence repo
            cadence_replacement = shlex.quote(Path.cwd().parent.absolute().resolve().as_posix())

        if flowgo_version:
            flowgo_replacement = f'github.com/onflow/flow-go@{flowgo_version}'
        else:
            # default: use the newest version of flow-go available
            flowgo_replacement = f'github.com/onflow/flow-go@latest'

        with cwd(working_dir / self.path):
            if prepare:
                logger.info("Editing dependencies")
                subprocess.run([
                    "go", "get", flowgo_replacement,
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

def load_index(path: Path) -> List[str]:
    logger.info(f"Loading suite index from {path} ...")
    with path.open(mode="r") as f:
        return yaml.safe_load(f)

@dataclass
class Description:
    description: str
    url: str
    branch: str
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
        go_test: bool,
        cadence_version: Optional[str],
        flowgo_version: Optional[str],
    ) -> (bool):

        working_dir = SUITE_PATH / name

        if prepare:
            self._clone(working_dir)

        go_tests_succeeded = True
        if go_test:
            for test in self.go_tests:
                if not test.run(working_dir, prepare=prepare, cadence_version=cadence_version, flowgo_version=flowgo_version):
                    go_tests_succeeded = False

        succeeded = go_tests_succeeded

        return succeeded

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
    "--cadence-version",
    default=None,
    help="version of Cadence for Go tests"
)
@click.option(
    "--flowgo-version",
    default=None,
    help="version of flow-go for Go tests"
)
@click.argument(
    "names",
    nargs=-1,
)
def main(
        rerun: bool,
        go_test: bool,
        cadence_version: str,
        flowgo_version: str,
        names: Collection[str]
):

    prepare = not rerun

    # Run for the current checkout

    current_success = run(
        prepare=prepare,
        go_test=go_test,
        cadence_version=cadence_version,
        flowgo_version=flowgo_version,
        names=names
    )

    if not current_success:
        exit(1)


def run(
        prepare: bool,
        go_test: bool,
        cadence_version: str,
        flowgo_version: str,
        names: Collection[str]
) -> (bool):

    all_succeeded = True

    logger.info(f'Chosen versions: cadence@{ cadence_version if cadence_version else "local version" }, flow-go@{flowgo_version if flowgo_version else "latest"}')

    if not names:
        names = load_index(SUITE_PATH / "index.yaml")

    for name in names:

        description = Description.load(name)

        run_succeeded = description.run(
            name,
            prepare=prepare,
            go_test=go_test,
            cadence_version=cadence_version,
            flowgo_version=flowgo_version,
        )

        if not run_succeeded:
            all_succeeded = False

    return all_succeeded


if __name__ == "__main__":
    logger = logging.getLogger(__name__)

    coloredlogs.install(
        level="INFO",
        fmt="%(asctime)s,%(msecs)03d %(levelname)s %(message)s"
    )

    main()
