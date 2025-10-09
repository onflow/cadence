# Source Compatibility Suite

The source compatibility suite prevents regressions and helps understand what impact changes in the language and implementation have on real-world Cadence projects.

The suite contains a [set of repository descriptions](https://github.com/onflow/cadence/tree/master/compat/suite). When the suite is run, the repositories get cloned and checked. The runner can optionally run benchmarks and compare against another commit / branch, producing output in the terminal (pretty), as JSON, or Markdown.

In the future we can integrate this as part of CI, maybe as a periodic job.

## Running

- Install the dependencies:

  ```sh
  pip3 install -r requirements.txt
  ```

- Run the suite:

  ```sh
  python3 main.py
  ```
