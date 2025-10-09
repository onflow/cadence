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
CLI_PATH = SUITE_PATH / "flow-cli" / "cmd" / "flow"

@contextmanager
def cwd(path):
    oldpwd = os.getcwd()
    os.chdir(path)
    try:
        yield
    finally:
        os.chdir(oldpwd)

def cadence_replacement(cadence_version: Optional[str]) -> str:
    if cadence_version:
        return f'github.com/onflow/cadence@{cadence_version}'
    # default: point to local cadence repo
    return shlex.quote(Path(__file__).parent.parent.absolute().resolve().as_posix())

def flowgo_replacement(flowgo_version: Optional[str]) -> str:
    if flowgo_version:
        return f'github.com/onflow/flow-go@{flowgo_version}'
    # default: use the newest version of flow-go available
    return 'github.com/onflow/flow-go@latest'

def build_cli(
    cadence_version: Optional[str],
    flowgo_version: Optional[str]
):
    logger.info("Building CLI ...")

    with cwd(SUITE_PATH):
        working_dir = SUITE_PATH / "flow-cli"
        Git.clone("https://github.com/onflow/flow-cli.git", "master", working_dir)
        with cwd(working_dir):
            Go.replace_dependencies(
                cadence_version=cadence_version,
                flowgo_version=flowgo_version
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
        flowgo_version: Optional[str]
    ) -> bool:

        with cwd(working_dir / self.path):
            if prepare:
                Go.replace_dependencies(cadence_version, flowgo_version)

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
        flowgo_version: Optional[str]
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
                    flowgo_version=flowgo_version
                ):
                    go_tests_succeeded = False

        cadence_tests_succeeded = True
        if cadence_test:
            for test in self.cadence_tests:
                if not test.run(
                    working_dir,
                    prepare=prepare,
                    cadence_version=cadence_version,
                    flowgo_version=flowgo_version
                ):
                    cadence_tests_succeeded = False

        return go_tests_succeeded and cadence_tests_succeeded

class Go:

    @staticmethod
    def mod_replace(original: str, replacement: str):
        subprocess.run(
            ["go", "mod", "edit", "-replace", f'{original}={replacement}'],
            check=True
        )

    @staticmethod
    def mod_tidy():
        subprocess.run(
            ["go", "mod", "tidy"],
            check=True
        )

    @staticmethod
    def replace_dependencies(
        cadence_version: Optional[str],
        flowgo_version: Optional[str]
    ):
        logger.info("Editing dependencies")
        Go.mod_replace("github.com/onflow/cadence", cadence_replacement(cadence_version))
        Go.mod_replace("github.com/onflow/flow-go", flowgo_replacement(flowgo_version))
        Go.mod_tidy()

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
    names: Collection[str]
):

    logger.info(
        f'Chosen versions: '
        + f'cadence@{ cadence_version if cadence_version else "local version" }, '
        + f'flow-go@{flowgo_version if flowgo_version else "latest"}'
    )

    prepare = not rerun

    if cadence_test and prepare:
        build_cli(
            cadence_version=cadence_version,
            flowgo_version=flowgo_version,
        )

    # Run for the current checkout

    current_success = run(
        prepare=prepare,
        go_test=go_test,
        cadence_test=cadence_test,
        cadence_version=cadence_version,
        flowgo_version=flowgo_version,
        names=names
    )

    if not current_success:
        exit(1)


def run(
    prepare: bool,
    go_test: bool,
    cadence_test: bool,
    cadence_version: str,
    flowgo_version: str,
    names: Collection[str]
) -> (bool):

    all_succeeded = True

    if not names:
        names = [f.stem for f in SUITE_PATH.glob("*.yaml")]

    for name in names:

        description = Description.load(name)

        run_succeeded = description.run(
            name,
            prepare=prepare,
            go_test=go_test,
            cadence_test=cadence_test,
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
