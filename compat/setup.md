
# Benchmark System Setup

To get reliable results for benchmarks, it is important to configure the system
to have a good **stable** performance, compared to good **peak** performance.
This ensures that there is little noise and variance in the results.

## CPU Frequency governor

- Use `performance` governor:

  ```sh
  cpupower -c all frequency-set -g performance
  ```

- Check with:

  1. `cpupower -c all frequency-info` should show

      ```
      available cpufreq governors: performance powersave
      current policy: frequency should be within 800 MHz and 3.10 GHz.
                      The governor "performance" may decide which speed to use
                      within this range.
      current CPU frequency: Unable to call hardware
      current CPU frequency: 3.10 GHz (asserted by call to kernel)
      ```

  2. `cat /sys/devices/system/cpu/*/cpufreq/scaling_governor` should show
      `performance` for all CPUs

- Disable in Systemd with:

  ```sh
  systemctl disable ondemand
  ```

## Turbo Boost state

- Disable using:

  ```sh
  /bin/echo 1 > /sys/devices/system/cpu/intel_pstate/no_turbo
  ```

  (If it has no effect, disable via BIOS)

- Check with:

  ```sh
  cpupower -c all frequency-info
  ```

  should show

  ```
  boost state support:
    Supported: no
    Active: no
  ```

## Hyper-Threading

- Disable using:

  ```sh
  echo off > /sys/devices/system/cpu/smt/control
  ```

- Check with:

  ```sh
  cat /sys/devices/system/cpu/smt/control
  ```

  should show `off`, `forceoff` (if disabled in BIOS), or `notsupported`

## Stop Systemd services

- `systemctl stop`

## Disable Address Space Randomization

- Disable using:

  ```sh
  echo 0 > /proc/sys/kernel/randomize_va_space
  ```

- Check with:

  ```sh
  cat /proc/sys/kernel/randomize_va_space
  ```

  should show `0`

## CPU affinity

- Activate shield using:

  ```sh
  cset shield --cpu 1-3 --kthread=on
  ```

- Check with:

  ```sh
  cset shield -s
  ```

  should show

  ```
  cset: "user" cpuset of CPUSPEC(1-3) with 0 tasks running
  cset: done
  ```

- Run commands with: `cset shield --exec <command> -- <args>`

## Verifying

Run the benchmarks with `--compare-ref HEAD` to compare against the same code.
Result deltas should be <2-3%.

## References

- https://easyperf.net/blog/2019/08/02/Perf-measurement-environment-on-Linux
- https://github.com/scala/scala-dev/issues/338
- https://github.com/scala/compiler-benchmark/blob/master/scripts/benv
- https://llvm.org/docs/Benchmarking.html
- https://pyperf.readthedocs.io/en/latest/system.html
- https://vstinner.github.io/journey-to-stable-benchmark-system.html
- https://developer.download.nvidia.com/video/gputechconf/gtc/2019/presentation/s9956-best-practices-when-benchmarking-cuda-applications_V2.pdf
- https://documentation.suse.com/sle-rt/12-SP4/html/SLE-RT-all/cha-shielding-model.html