# Source Compatibility Suite

## Running

- Install the dependencies:

  ```sh
  pip3 install -r requirements.txt
  ```

- Run the suite. For example, to clone the repositories, benchmark, and compare to branch `master`:

  ```sh
  python3 main.py --format=pretty --bench --compare-ref master
  ```
