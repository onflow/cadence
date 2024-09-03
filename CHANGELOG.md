<a name="v1.0.0-preview.52"></a>
# [v1.0.0-preview.52](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.52) - 31 Aug 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.52 -->

## What's Changed
### 🛠 Improvements
* Improve address decoding error by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3560
### 🐞 Bug Fixes
* Handle unmigrated path capabilities and path links in Account.capabilities functions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3562
### 🧪 Testing
* Add contract-update test for removing a field from nested resource by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3559
* Add test for modifying events during contract updates by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3561
### Other Changes
* Merge `release/v1.0.0-preview.51` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3558


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.51...v1.0.0-preview.52

[Changes][v1.0.0-preview.52]


<a name="v1.0.0-preview.51"></a>
# [v1.0.0-preview.51](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.51) - 29 Aug 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.51 -->

## What's Changed
### ⭐ Features
* Add sema.Access.QualifiedKeyword by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3552
### 🐞 Bug Fixes
* Fix Cadence Parser NPM Package by [@bluesign](https://github.com/bluesign) in https://github.com/onflow/cadence/pull/3551
* Fix contract update error when old program has errors by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3554
### 🧪 Testing
* Add regression tests for nil-coalescing bug by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3553
* Smoke test update of enum key in dictionary by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3556
### 📖 Documentation
* Staged contracts report Aug 27 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3557
### Other Changes
* Merge `release/v1.0.0-preview.50` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3549


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.50...v1.0.0-preview.51

[Changes][v1.0.0-preview.51]


<a name="v1.0.0-preview.50"></a>
# [v1.0.0-preview.50](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.50) - 23 Aug 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.50 -->

## What's Changed
### 🐞 Bug Fixes
* Fix program recovery by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3548
### 📖 Documentation
* Add mainnet migration report for Aug 21 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3547
### Other Changes
* Merge `release/v1.0.0-preview.49` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3545


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.49...v1.0.0-preview.50

[Changes][v1.0.0-preview.50]


<a name="v1.0.0-preview.49"></a>
# [v1.0.0-preview.49](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.49) - 21 Aug 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.49 -->

## What's Changed
### ⭐ Features
* Add `Type.isRecovered` field by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3505
### 🛠 Improvements
* Fix contract update validation: Check updated contract name by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3530
* Check if CBOR tag number is reserved by atree before using it to encode Cadence values by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3532
### 🐞 Bug Fixes
* Add support for exporting unmigrated path capabilities by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3538
* Handle invalid index-expressions in view assignment by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3539
* Make capabilities iteration ordered by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3542
### Other Changes
* add staged report for aug 12 by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3525
* Merge `release/v1.0.0-preview.48` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3526
* Add staged contracts report for testnet spork on aug 14 by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3531
* Crescendo Mainnet migration report Aug 16 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3541


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.48...v1.0.0-preview.49

[Changes][v1.0.0-preview.49]


<a name="v1.0.0-preview.48"></a>
# [v1.0.0-preview.48](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.48) - 13 Aug 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.48 -->

## What's Changed
### 🛠 Improvements
* Add capability stored path info to the report by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3524
### Other Changes
* Merge `release/v1.0.0-preview.47` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3523


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.47...v1.0.0-preview.48

[Changes][v1.0.0-preview.48]


<a name="v1.0.0-preview.47"></a>
# [v1.0.0-preview.47](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.47) - 13 Aug 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.47 -->

## What's Changed
### 🛠 Improvements
* Improve storage path capability migration reporting by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3522
### Other Changes
* Merge `release/v1.0.0-preview.46` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3521


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.46...v1.0.0-preview.47

[Changes][v1.0.0-preview.47]


<a name="v1.0.0-preview.46"></a>
# [v1.0.0-preview.46](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.46) - 12 Aug 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.46 -->

## What's Changed
### 🛠 Improvements
* Infer missing borrow type for storage capabilities by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3519
* Improve borrow type inferrence for storage path capabilities by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3520
### Other Changes
* Merge `release/v1.0.0-preview.45` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3518


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.45...v1.0.0-preview.46

[Changes][v1.0.0-preview.46]


<a name="v1.0.0-preview.45"></a>
# [v1.0.0-preview.45](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.45) - 12 Aug 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.45 -->

## What's Changed
### 🛠 Improvements
* Improve storage capability migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3516
### 🐞 Bug Fixes
* Avoid conversion from sema-type to StaticType in storage cap migration by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3517
### Other Changes
* Merge `release/v1.0.0-preview.44` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3515


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.44...v1.0.0-preview.45

[Changes][v1.0.0-preview.45]


<a name="v1.0.0-preview.44"></a>
# [v1.0.0-preview.44](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.44) - 09 Aug 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.44 -->

## What's Changed
### 🐞 Bug Fixes
* Report and skip storage caps with missing borrow type by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3514
### Other Changes
* Update staged-contracts-report-2024-08-07T10-00-00Z-testnet.md by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3513
* Merge `release/v1.0.0-preview.43` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3512


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.43...v1.0.0-preview.44

[Changes][v1.0.0-preview.44]


<a name="v1.0.0-preview.43"></a>
# [v1.0.0-preview.43](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.43) - 08 Aug 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.43 -->

## What's Changed
### 🐞 Bug Fixes
* Fix migration of storage path capabilities by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3511
### Other Changes
* Merge `release/v1.0.0-preview.42` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3504
* adding migration report from Testnet migration on 31st July by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3506
* Add migration report for aug 7 by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3510


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.42...v1.0.0-preview.43

[Changes][v1.0.0-preview.43]


<a name="v1.0.0-preview.42"></a>
# [v1.0.0-preview.42](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.42) - 31 Jul 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.42 -->

## What's Changed
### 🛠 Improvements
* Migrate capability values targeting storage paths by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3503


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.41...v1.0.0-preview.42

[Changes][v1.0.0-preview.42]


<a name="v1.0.0-preview.41"></a>
# [v1.0.0-preview.41](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.41) - 29 Jul 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.41 -->

## What's Changed
### ⭐ Features
* Handle broken contracts by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3482
* Merge atree register inlining v1.0 by [@fxamacker](https://github.com/fxamacker)  in https://github.com/onflow/cadence/pull/3502
### 🛠 Improvements
* Use atree readonly iterators when mutation of values is not needed by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3170
* Atree register inlining for v1.0 by [@fxamacker](https://github.com/fxamacker)  in https://github.com/onflow/cadence/pull/3048
* Improve atree register inlining v1.0 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3230
### 🐞 Bug Fixes
* Fix Cadence 1.0 migration of dictionary values when using atree inlined data by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3316
* Fix EntitlementDeclaration EndPos by [@jribbink](https://github.com/jribbink) in https://github.com/onflow/cadence/pull/3498
### 🧪 Testing
* Add reproducer as test for Cadence 1.0 migration issue [#3288](https://github.com/onflow/cadence/issues/3288) by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3314
### Other Changes
* Bump atree version in Cadence v1.0 for new features (e.g. deduplication of Cadence dictionary type info)  by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3178
* Sync atree inlining branch with master by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3179
* Merge master into v1.0 atree register inlining feature branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3229
* Bump onflow/atree to latest version in feature/atree-register-inlining-v1.0 by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3223
* Bump atree version in feature/atree-register-inlining-v1.0 by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3256
* Update atree register inlining 1.0 feature branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3259
* Merge master into atree register inlining v1.0 branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3270
* Bump atree version in feature/atree-register-inlining-v1.0 by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3295
* Update atree inlining v1.0 feature branch  by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3301
* Update atree inlining 1.0 branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3306
* Update atree register inlining v1.0 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3320
* Bump atree version to v0.8.0-rc.2 in feature/atree-register-inlining-v1.0 by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3346
* Sync feature/atree-register-inlining-v1.0 with master by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3351
* Update atree register inlining v1.0 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3359
* Bump atree version to v0.8.0-rc.3 in feature/atree-register-inlining-v1.0 by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3365
* Update atree register inlining v1.0 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3378
* Update atree register inlining feature branch to v1.0.0-preview.30 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3383
* Update atree inlining branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3390
* Update feature/atree-register-inlining-v1.0 to latest master by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3414
* Merge v1.0.0-preview.34 into feature/atree-register-inlining-v1.0 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3416
* Update atree inlining cadence v1.0 feature branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3438
* Update atree register inlining v1.0 branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3454
* Update atree inlining cadence v1.0 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3467
* Update atree inlining branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3472
* Update feature/atree-register-inlining-v1.0 by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3474
* Update atree register inlining 1.0 branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3488
* Merge `release/v1.0.0-preview.40` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3496
* Update atree inlining v1.0 branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3501
* Update feature/atree-register-inlining-v1.0 to use atree v0.8.0-rc.5 by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3500


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.40...v1.0.0-preview.41

[Changes][v1.0.0-preview.41]


<a name="v1.0.0-preview.40"></a>
# [v1.0.0-preview.40](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.40) - 26 Jul 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.40 -->

## What's Changed
### ⭐ Features
* Add stack traces back to unexpected errors by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3492
### 🐞 Bug Fixes
* Fix supertype inference by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3493
* Return user error when CCF encodes attachment field by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3494
### 🧪 Testing
* Add checker tests for nil-coalesce type inference by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3495
### Other Changes
* Merge `release/v1.0.0-preview.39` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3487
* adding migration result from Jul 24, preview 34 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3489
* Move migration report files for 2024-07-17 into correct directory by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3490


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.39...v1.0.0-preview.40

[Changes][v1.0.0-preview.40]


<a name="v1.0.0-preview.39"></a>
# [v1.0.0-preview.39](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.39) - 24 Jul 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.39 -->

## What's Changed
### ⭐ Features
* Add an empty implementation of `runtime.Interface` by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3485
### 🛠 Improvements
* Improve intersected type error message by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3483
* Bring back members of path capability value by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3486
* Cleanup `ConvertSemaAccessToStaticAuthorization` method by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3479
### 🐞 Bug Fixes
* Fix data races caused by empty string singleton by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3481
### 🧪 Testing
* Update Fungible token transfer benchmark by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3471
### Other Changes
* Merge `release/v1.0.0-preview.38` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3470
* adding report for TN migration Jul 17 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3476


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.38...v1.0.0-preview.39

[Changes][v1.0.0-preview.39]


<a name="v1.0.0-preview-atree-register-inlining.39"></a>
# [v1.0.0-preview-atree-register-inlining.39](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview-atree-register-inlining.39) - 24 Jul 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview-atree-register-inlining.39 -->

## What's Changed
### ⭐ Features
* Add an empty implementation of `runtime.Interface` by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3485
### 🛠 Improvements
* Improve intersected type error message by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3483
* Bring back members of path capability value by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3486
* Cleanup `ConvertSemaAccessToStaticAuthorization` method by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3479
### 🐞 Bug Fixes
* Fix data races caused by empty string singleton by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3481
### 🧪 Testing
* Update Fungible token transfer benchmark by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3471
### Other Changes
* Update feature/atree-register-inlining-v1.0 by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3474
* adding report for TN migration Jul 17 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3476
* Merge `release/v1.0.0-preview.39` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3487
* Update atree register inlining 1.0 branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3488


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.38...v1.0.0-preview-atree-register-inlining.39

[Changes][v1.0.0-preview-atree-register-inlining.39]


<a name="v1.0.0-preview.38"></a>
# [v1.0.0-preview.38](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.38) - 17 Jul 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.38 -->

## What's Changed
### 🛠 Improvements
* Speed up update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3468
* Changed data type of account key index to uin32 by [@janezpodhostnik](https://github.com/janezpodhostnik) in https://github.com/onflow/cadence/pull/3465
* Improve Cadence composite to Go struct decoding by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3469
### Other Changes
* Merge `release/v1.0.0-preview.37` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3466


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.37...v1.0.0-preview.38

[Changes][v1.0.0-preview.38]


<a name="v1.0.0-preview-atree-register-inlining.38"></a>
# [v1.0.0-preview-atree-register-inlining.38](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview-atree-register-inlining.38) - 16 Jul 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview-atree-register-inlining.38 -->

## What's Changed
### 🛠 Improvements
* Speed up update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3468
* Changed data type of account key index to uin32 by [@janezpodhostnik](https://github.com/janezpodhostnik) in https://github.com/onflow/cadence/pull/3465
* Improve Cadence composite to Go struct decoding by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3469
### Other Changes
* Merge `release/v1.0.0-preview.38` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3470
* Update atree inlining branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3472


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.37...v1.0.0-preview-atree-register-inlining.38

[Changes][v1.0.0-preview-atree-register-inlining.38]


<a name="v1.0.0-preview.37"></a>
# [v1.0.0-preview.37](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.37) - 12 Jul 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.37 -->

## What's Changed
### ⭐ Features
* Add a String.contains function by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3455
* Add String.index and String.count, fix grapheme boundary functions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3456
* Emit events when capability controllers are issued by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3460
* Emit events for more Capability Controller and capability operations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3464
### 🐞 Bug Fixes
* Fix String.split by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3457
* Fix String.replaceAll by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3458
* Fix account link migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3461
### 🧪 Testing
* Add some more tests for string functions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3453
### 📖 Documentation
* add staged contracts report 07-10 by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3462
* add root block for migration 07-10 by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3463
### Other Changes
* Merge `release/v1.0.0-preview.36` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3452


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.36...v1.0.0-preview.37

[Changes][v1.0.0-preview.37]


<a name="v1.0.0-preview-atree-register-inlining.37"></a>
# [v1.0.0-preview-atree-register-inlining.37](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview-atree-register-inlining.37) - 12 Jul 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview-atree-register-inlining.37 -->

## What's Changed
### ⭐ Features
* Add a String.contains function by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3455
* Add String.index and String.count, fix grapheme boundary functions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3456
* Emit events when capability controllers are issued by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3460
* Emit events for more Capability Controller and capability operations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3464
### 🐞 Bug Fixes
* Fix String.split by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3457
* Fix String.replaceAll by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3458
* Fix account link migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3461
### 🧪 Testing
* Add some more tests for string functions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3453
### 📖 Documentation
* add staged contracts report 07-10 by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3462
* add root block for migration 07-10 by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3463
### Other Changes
* Merge `release/v1.0.0-preview.36` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3452
* Merge `release/v1.0.0-preview.37` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3466
* Update atree inlining cadence v1.0 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3467


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.36...v1.0.0-preview-atree-register-inlining.37

[Changes][v1.0.0-preview-atree-register-inlining.37]


<a name="v1.0.0-preview.36"></a>
# [v1.0.0-preview.36](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.36) - 08 Jul 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.36 -->

## What's Changed
### 🛠 Improvements
* Improve the commit-based version handling in the update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3440
* Improve update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3441
* Improve handling of releases in update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3442
* Allow borrowing of capability with subtype by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3449
* Simplify subtyping by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3447
### 🐞 Bug Fixes
* Fix toConstantSized by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3446
### 📖 Documentation
* Remove mention of feature branch by [@chasefleming](https://github.com/chasefleming) in https://github.com/onflow/cadence/pull/3444
* add 2024-07-03 staged contracts report by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3450
### Other Changes
* Merge `release/v1.0.0-preview.35` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3437
* add migrations data for 2024-06-26 by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3439
* add root block info for migration net by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3443
* Add root block info for migration 2024-07-03 by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3451

## New Contributors
* [@chasefleming](https://github.com/chasefleming) made their first contribution in https://github.com/onflow/cadence/pull/3444

**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.35...v1.0.0-preview.36

[Changes][v1.0.0-preview.36]


<a name="v1.0.0-preview-atree-register-inlining.36"></a>
# [v1.0.0-preview-atree-register-inlining.36](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview-atree-register-inlining.36) - 08 Jul 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview-atree-register-inlining.36 -->

## What's Changed
### 🛠 Improvements
* Improve the commit-based version handling in the update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3440
* Improve update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3441
* Improve handling of releases in update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3442
* Allow borrowing of capability with subtype by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3449
* Simplify subtyping by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3447
### 🐞 Bug Fixes
* Fix toConstantSized by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3446
### 📖 Documentation
* Remove mention of feature branch by [@chasefleming](https://github.com/chasefleming) in https://github.com/onflow/cadence/pull/3444
* add 2024-07-03 staged contracts report by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3450
### Other Changes
* Merge `release/v1.0.0-preview.35` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3437
* add migrations data for 2024-06-26 by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3439
* add root block info for migration net by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3443
* Add root block info for migration 2024-07-03 by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3451
* Update atree register inlining v1.0 branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3454

## New Contributors
* [@chasefleming](https://github.com/chasefleming) made their first contribution in https://github.com/onflow/cadence/pull/3444

**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.35...v1.0.0-preview-atree-register-inlining.36

[Changes][v1.0.0-preview-atree-register-inlining.36]


<a name="v1.0.0-preview-atree-register-inlining.35"></a>
# [v1.0.0-preview-atree-register-inlining.35](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview-atree-register-inlining.35) - 25 Jun 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview-atree-register-inlining.35 -->

## What's Changed
### 💥 Language Breaking Changes
* Import contracts as references by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3417
### 💥 Go API Breaking Chance
* Remove access to field slices from composite and interface types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3432
### ⭐ Features
* Add command to dump all hard keywords by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3431
### 🛠 Improvements
* Ensure contracts cannot be borrowed with an authorization by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3421
* Improve dump-builtin-types command: Handle parameterized types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3425
* Improve InclusiveRange type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3426
### 🐞 Bug Fixes
* Fix JSON output by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3413
* Improve parsing of commit from Go's pseudo-version in update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3418
* Simplify nil-coalescing checking by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3423
### 📖 Documentation
* Update staged-contracts-report-2024-06-12T13-03-00Z-testnet.md by [@vishalchangrani](https://github.com/vishalchangrani) in https://github.com/onflow/cadence/pull/3429
### Other Changes
* adding report from June 12 TN state migration  by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3420
* Update update tool config by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3427
* adding report for Testnet migration on June 19 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3433
* Update update tool config by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3435
* Update atree inlining cadence v1.0 feature branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3438


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.34...v1.0.0-preview-atree-register-inlining.35

[Changes][v1.0.0-preview-atree-register-inlining.35]


<a name="v1.0.0-preview.35"></a>
# [v1.0.0-preview.35](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.35) - 24 Jun 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.35 -->

## What's Changed
### 💥 Language Breaking Changes
* Import contracts as references by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3417
### 💥 Go API Breaking Chance
* Remove access to field slices from composite and interface types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3432
### ⭐ Features
* Add command to dump all hard keywords by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3431
### 🛠 Improvements
* Ensure contracts cannot be borrowed with an authorization by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3421
* Improve dump-builtin-types command: Handle parameterized types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3425
* Improve InclusiveRange type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3426
### 🐞 Bug Fixes
* Fix JSON output by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3413
* Improve parsing of commit from Go's pseudo-version in update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3418
* Simplify nil-coalescing checking by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3423
### 📖 Documentation
* Update staged-contracts-report-2024-06-12T13-03-00Z-testnet.md by [@vishalchangrani](https://github.com/vishalchangrani) in https://github.com/onflow/cadence/pull/3429
### Other Changes
* Merge `release/v1.0.0-preview.34` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3415
* adding report from June 12 TN state migration  by [@zhangchiqing](https://github.com/zhangchiqing) in https://github.com/onflow/cadence/pull/3420
* Update update tool config by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3427
* adding report for Testnet migration on June 19 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3433
* Update update tool config by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3435


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.34...v1.0.0-preview.35

[Changes][v1.0.0-preview.35]


<a name="v0.42.12"></a>
# [v0.42.12](https://github.com/onflow/cadence/releases/tag/v0.42.12) - 13 Jun 2024

<!-- Release notes generated using configuration in .github/release.yml at v0.42.12 -->

## What's Changed
### 🐞 Bug Fixes
* Fix JSON output by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3412
* Simplify nil-coalescing checking by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3422
### Other Changes
* Merge `release/v0.42.11` to `v0.42` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3333


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.42.11...v0.42.12

[Changes][v0.42.12]


<a name="v1.0.0-preview.34"></a>
# [v1.0.0-preview.34](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.34) - 12 Jun 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.34 -->

## What's Changed
### 🛠 Improvements
* Put feature that allows type removals in contract updates behind a feature flag, disabled by default by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3410
* Also migrate path-capability and account-capability storage maps by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3411
### 🐞 Bug Fixes
* Prevent storage reference to another reference by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3404
* Add access top type to model inaccessible access for identity maps by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3406
### Other Changes
* Merge `release/v1.0.0-preview.33` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3399
* adding report from June 5 TN state migration by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3400


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.33...v1.0.0-preview.34

[Changes][v1.0.0-preview.34]


<a name="v1.0.0-preview-atree-register-inlining.34"></a>
# [v1.0.0-preview-atree-register-inlining.34](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview-atree-register-inlining.34) - 12 Jun 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview-atree-register-inlining.34 -->

## What's Changed
### 🛠 Improvements
* Remove stacktrace from errors by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3392
* Improve update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3395
* Cache results of static types migration and entitlements migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3396
* Put feature that allows type removals in contract updates behind a feature flag, disabled by default by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3410
* Also migrate path-capability and account-capability storage maps by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3411
### 🐞 Bug Fixes
* Prevent storage reference to another reference by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3404
* Add access top type to model inaccessible access for identity maps by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3406
### Other Changes
* Merge `release/v1.0.0-preview.32` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3389
* Update copyright notice by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3391
* Update to Go 1.22 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3393
* Merge `release/v1.0.0-preview.33` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3399
* adding report from June 5 TN state migration by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3400
* Update feature/atree-register-inlining-v1.0 to latest master by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3414
* Merge `release/v1.0.0-preview.34` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3415
* Merge v1.0.0-preview.34 into feature/atree-register-inlining-v1.0 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3416


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.32...v1.0.0-preview-atree-register-inlining.34

[Changes][v1.0.0-preview-atree-register-inlining.34]


<a name="v1.0.0-preview.33"></a>
# [v1.0.0-preview.33](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.33) - 05 Jun 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.33 -->

## What's Changed
### 🛠 Improvements
* Remove stacktrace from errors by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3392
* Improve update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3395
* Cache results of static types migration and entitlements migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3396
### Other Changes
* Merge `release/v1.0.0-preview.32` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3389
* Update copyright notice by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3391
* Update to Go 1.22 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3393


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.32...v1.0.0-preview.33

[Changes][v1.0.0-preview.33]


<a name="v1.0.0-preview.32"></a>
# [v1.0.0-preview.32](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.32) - 04 Jun 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.32 -->

## What's Changed
### 🛠 Improvements
* Decrease move operator precedence relative to cast operators by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3368
* Improve dictionary key migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3386
### 🐞 Bug Fixes
* Don't short circuit references on member access when entitlements are involved by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3387
### Other Changes
* Merge `release/v1.0.0-preview.31` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3382
* Add report from May 29 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3384


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.31...v1.0.0-preview.32

[Changes][v1.0.0-preview.32]


<a name="v1.0.0-preview-atree-register-inlining.32"></a>
# [v1.0.0-preview-atree-register-inlining.32](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview-atree-register-inlining.32) - 04 Jun 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview-atree-register-inlining.32 -->

## What's Changed
### 🛠 Improvements
* Forbid interface removals by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3380
* Improve dictionary value migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3381
* Decrease move operator precedence relative to cast operators by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3368
* Improve dictionary key migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3386
### 🐞 Bug Fixes
* Fix error message by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3379
* Don't short circuit references on member access when entitlements are involved by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3387
### Other Changes
* Merge `release/v1.0.0-preview.30` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3377
* Merge `release/v1.0.0-preview.31` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3382
* Add report from May 29 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3384
* Update atree register inlining feature branch to v1.0.0-preview.30 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3383
* Update atree inlining branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3390


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.30...v1.0.0-preview-atree-register-inlining.32

[Changes][v1.0.0-preview-atree-register-inlining.32]


<a name="v1.0.0-preview.31"></a>
# [v1.0.0-preview.31](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.31) - 29 May 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.31 -->

## What's Changed
### 🛠 Improvements
* Forbid interface removals by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3380
* Improve dictionary value migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3381
### 🐞 Bug Fixes
* Fix error message by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3379
### Other Changes
* Merge `release/v1.0.0-preview.30` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3377


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.30...v1.0.0-preview.31

[Changes][v1.0.0-preview.31]


<a name="v1.0.0-preview.30"></a>
# [v1.0.0-preview.30](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.30) - 28 May 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.30 -->

## What's Changed
### ⭐ Features
* Allow types to be removed in contract updates by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3376
### 🛠 Improvements
* Fix string formatting for values by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3370
* Update the storage explorer to the latest version of flow-go by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3374
### 🐞 Bug Fixes
* Add deprecated path Capability into backwards compability by [@ianthpun](https://github.com/ianthpun) in https://github.com/onflow/cadence/pull/3373
* Remove incorrect caching from migrations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3375
### Other Changes
* Merge `release/v1.0.0-preview.29` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3358
* Uploading migration results for TN state snapshot taked May 13 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3360
* move may15 staged contracts report to the correct directory by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3361
* Adding migration report - TN May 22 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3367
* Add migration environment endpoint to the report by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3369


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.29...v1.0.0-preview.30

[Changes][v1.0.0-preview.30]


<a name="v1.0.0-preview-atree-register-inlining.30"></a>
# [v1.0.0-preview-atree-register-inlining.30](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview-atree-register-inlining.30) - 28 May 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview-atree-register-inlining.30 -->

## What's Changed
### ⭐ Features
* Allow types to be removed in contract updates by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3376
### 🛠 Improvements
* Fix string formatting for values by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3370
* Update the storage explorer to the latest version of flow-go by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3374
### 🐞 Bug Fixes
* Add deprecated path Capability into backwards compability by [@ianthpun](https://github.com/ianthpun) in https://github.com/onflow/cadence/pull/3373
* Remove incorrect caching from migrations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3375
### Other Changes
* Uploading migration results for TN state snapshot taked May 13 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3360
* move may15 staged contracts report to the correct directory by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3361
* Bump atree version to v0.8.0-rc.3 in feature/atree-register-inlining-v1.0 by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3365
* Adding migration report - TN May 22 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3367
* Add migration environment endpoint to the report by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3369
* Update atree register inlining v1.0 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3378


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.29...v1.0.0-preview-atree-register-inlining.30

[Changes][v1.0.0-preview-atree-register-inlining.30]


<a name="v1.0.0-preview.29"></a>
# [v1.0.0-preview.29](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.29) - 17 May 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.29 -->

## What's Changed
### 💥 Language Breaking Changes
* Return a copy for enum-typed members, during member-access via references by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3357
### Other Changes
* Merge `release/v1.0.0-preview.28` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3353


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.28...v1.0.0-preview.29

[Changes][v1.0.0-preview.29]


<a name="v1.0.0-preview-atree-register-inlining.29"></a>
# [v1.0.0-preview-atree-register-inlining.29](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview-atree-register-inlining.29) - 17 May 2024

<!-- Release notes generated using configuration in .github/release.yml at feature/atree-register-inlining-v1.0 -->

## What's Changed
### 💥 Language Breaking Changes
* Return a copy for enum-typed members, during member-access via references by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3357
### Other Changes
* Merge `release/v1.0.0-preview.29` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3358
* Update atree register inlining v1.0 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3359


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.28...v1.0.0-preview-atree-register-inlining.29

[Changes][v1.0.0-preview-atree-register-inlining.29]


<a name="v1.0.0-preview.28"></a>
# [v1.0.0-preview.28](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.28) - 15 May 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.28 -->

## What's Changed
### 🐞 Bug Fixes
* Fix restricted type checking in contract-update-validator by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3352
### Other Changes
* Merge `release/v1.0.0-preview.27` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3349


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.27...v1.0.0-preview.28

[Changes][v1.0.0-preview.28]


<a name="v1.0.0-preview.27"></a>
# [v1.0.0-preview.27](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.27) - 15 May 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.27 -->

## What's Changed
### 💥 Language Breaking Changes
* Use parent to determine whether an expression is target of an invocation by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3340
* Improve static validation of contract/transaction moves by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3341
### ⭐ Features
* Add Storage.NondeterministicCommit for faster migrations by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3348
### 🛠 Improvements
* Make migration error stacktrace configurable by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3322
* Add/improve caching in static type and entitlements migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3329
* Allow construction of paths with syntactically invalid identifiers by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3338
* Add invalidation for storage references by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3343
### 🐞 Bug Fixes
* Remove invalid ID capability error by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3325
* Fix borrowing of contract interface by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3327
* Port bug fixes by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3331
* Handle missing interfaces in Cadence 1.0 contract update validator by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3337
* Check dictionary keys when checking the ability for dereferencing by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3339
* Wrap host-function typed member-values with a bound-function by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3342
### 🧪 Testing
* Add back transaction `execute` tests by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3345
### Other Changes
* Merge `release/v1.0.0-preview.26` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3321
* adding staged contracts migration report from TN state taken on May 8 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3330
* Update flow-go related configuration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3319
* Bump atree version to v0.7.0-rc.2 in master branch by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3347


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.26...v1.0.0-preview.27

[Changes][v1.0.0-preview.27]


<a name="v1.0.0-preview-atree-register-inlining.28"></a>
# [v1.0.0-preview-atree-register-inlining.28](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview-atree-register-inlining.28) - 15 May 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview-atree-register-inlining.28 -->

## What's Changed
### 🐞 Bug Fixes
* Fix restricted type checking in contract-update-validator by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3352
### Other Changes
* Merge `release/v1.0.0-preview.28` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3353


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.27...v1.0.0-preview-atree-register-inlining.28

[Changes][v1.0.0-preview-atree-register-inlining.28]


<a name="v1.0.0-preview-atree-register-inlining.27"></a>
# [v1.0.0-preview-atree-register-inlining.27](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview-atree-register-inlining.27) - 15 May 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview-atree-register-inlining.27 -->

## What's Changed
### 💥 Language Breaking Changes
* Use parent to determine whether an expression is target of an invocation by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3340
* Improve static validation of contract/transaction moves by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3341
### ⭐ Features
* Add Storage.NondeterministicCommit for faster migrations by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3348
### 🛠 Improvements
* Make migration error stacktrace configurable by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3322
* Add/improve caching in static type and entitlements migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3329
* Allow construction of paths with syntactically invalid identifiers by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3338
* Add invalidation for storage references by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3343
### 🐞 Bug Fixes
* Remove invalid ID capability error by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3325
* Fix borrowing of contract interface by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3327
* Port bug fixes by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3331
* Handle missing interfaces in Cadence 1.0 contract update validator by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3337
* Check dictionary keys when checking the ability for dereferencing by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3339
* Wrap host-function typed member-values with a bound-function by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3342
### 🧪 Testing
* Add back transaction `execute` tests by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3345
### Other Changes
* Merge `release/v1.0.0-preview.26` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3321
* adding staged contracts migration report from TN state taken on May 8 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3330
* Update flow-go related configuration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3319
* Bump atree version to v0.8.0-rc.2 in feature/atree-register-inlining-v1.0 by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3346
* Bump atree version to v0.7.0-rc.2 in master branch by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3347
* Merge `release/v1.0.0-preview.27` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3349
* Sync feature/atree-register-inlining-v1.0 with master by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3351


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.26...v1.0.0-preview-atree-register-inlining.27

[Changes][v1.0.0-preview-atree-register-inlining.27]


<a name="v0.42.11"></a>
# [v0.42.11](https://github.com/onflow/cadence/releases/tag/v0.42.11) - 09 May 2024

<!-- Release notes generated using configuration in .github/release.yml at v0.42.11 -->

## What's Changed
### 🐞 Bug Fixes
* Port bug fixes to v0.42 by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3332
### Other Changes
* Merge `release/v0.42.10` to `v0.42` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3238


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.42.10...v0.42.11

[Changes][v0.42.11]


<a name="v1.0.0-preview.26"></a>
# [v1.0.0-preview.26](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.26) - 07 May 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.26 -->

## What's Changed
### ⭐ Features
* Add a command to generate a mermaid diagram by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3307
### 🛠 Improvements
* Make static type cache an interface by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3315
* Improve dictionary migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3318
### 🧪 Testing
* Add test for untyped optional nested-reference creation by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3317
### Other Changes
* Merge `release/v1.0.0-preview.25` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3305
* Adding migration report for TN state snapshot taken May 1st by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3311


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.25...v1.0.0-preview.26

[Changes][v1.0.0-preview.26]


<a name="v1.0.0-preview-atree-register-inlining.26"></a>
# [v1.0.0-preview-atree-register-inlining.26](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview-atree-register-inlining.26) - 07 May 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview-atree-register-inlining.26 -->

## What's Changed
### ⭐ Features
* Add a command to generate a mermaid diagram by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3307
### 🛠 Improvements
* Make static type cache an interface by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3315
* Improve dictionary migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3318
### 🐞 Bug Fixes
* Fix Cadence 1.0 migration of dictionary values when using atree inlined data by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3316
### 🧪 Testing
* Add test for untyped optional nested-reference creation by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3317
* Add reproducer as test for Cadence 1.0 migration issue [#3288](https://github.com/onflow/cadence/issues/3288) by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3314
### Other Changes
* Adding migration report for TN state snapshot taken May 1st by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3311
* Update atree register inlining v1.0 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3320


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.25...v1.0.0-preview-atree-register-inlining.26

[Changes][v1.0.0-preview-atree-register-inlining.26]


<a name="v1.0.0-preview-atree-register-inlining.25"></a>
# [v1.0.0-preview-atree-register-inlining.25](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview-atree-register-inlining.25) - 02 May 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview-atree-register-inlining.25 -->

## What's Changed
### 💥 Language Breaking Changes
* Produce a validator error when contracts would produce unrepresentable entitlements sets in the migration by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3304
### Other Changes
* Update atree inlining 1.0 branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3306


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.24...v1.0.0-preview-atree-register-inlining.25

[Changes][v1.0.0-preview-atree-register-inlining.25]


<a name="v1.0.0-preview.25"></a>
# [v1.0.0-preview.25](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.25) - 02 May 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.25 -->

## What's Changed
### 💥 Language Breaking Changes
* Produce a validator error when contracts would produce unrepresentable entitlements sets in the migration by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3304


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.24...v1.0.0-preview.25

[Changes][v1.0.0-preview.25]


<a name="v1.0.0-preview-atree-register-inlining.24"></a>
# [v1.0.0-preview-atree-register-inlining.24](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview-atree-register-inlining.24) - 01 May 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview-atree-register-inlining.24 -->

## What's Changed
### 💥 Go API Breaking Chance
* Unexport fields to prevent access by index by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3290
* Remove Value.ToGoValue and NewValue by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3291
### 🛠 Improvements
* Add runtime check for transaction value moves by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3298
* Add predicate function and test for atree's PersistentSlabStorage.FixLoadedBrokenReferences by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3300
### Other Changes
* Merge `release/v1.0.0-preview.23` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3289
* Update update tool config by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3292
* Bump atree version in master branch by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3293
* Bump atree version in feature/atree-register-inlining-v1.0 by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3295
* Update atree inlining v1.0 feature branch  by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3301


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.23...v1.0.0-preview-atree-register-inlining.24

[Changes][v1.0.0-preview-atree-register-inlining.24]


<a name="v1.0.0-preview.24"></a>
# [v1.0.0-preview.24](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.24) - 30 Apr 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.24 -->

## What's Changed
### 💥 Go API Breaking Chance

The API of the Go package `cadence` improved. These changes only impact users of the Go API of Cadence. These changes had no impact on the language itself or programs written in Cadence.

- It is no longer possible to access composite fields (`cadence.Composite` types, like `cadence.Struct`, `cadence.Resource`, etc) by index. The `Fields` field got unexported.

  <details>
  <summary>Details:</summary>

    - Accessing fields by index makes code dependent on the order of fields in the Cadence type definition, which may change (order is insignificant)
    - The order of fields of composite values returned from the chain (e.g. from a script) is planned to be made deterministic in the future. The Go API change prepares for this upcoming encoding change. For more details see https://github.com/onflow/cadence/issues/2952
    - Accessing fields by name improves code, as it removes possibilities for extracting field values incorrectly (by wrong index)
    - There are two ways to get the value of a composite field:
        - If multiple field values are needed from the composite value, 
        use `cadence.FieldsMappedByName`, which returns a `map[string]cadence.Value`.
            
            For example:
            
            ```go
            const fooTypeSomeIntFieldName = "someInt"
            const fooTypeSomeStringFieldName = "someString"
            
            // decodeFooEvent decodes the Cadence event:
            //
            //	event Foo(someInt: Int, someString: String)
            //
            // It returns an error if the event does not have the expected fields.
            func decodeFooEvent(event cadence.Event) (someInt cadence.Int, someString cadence.String, err error) {
            	fields := cadence.FieldsMappedByName(event)
            
            	var ok bool
            
            	someIntField := fields[fooTypeSomeIntFieldName]
            	someInt, ok = someIntField.(cadence.Int)
            	if !ok {
            		return cadence.Int{}, "", fmt.Errorf("wrong field type: expected Int, got %T", someInt)
            	}
            	
            	someStringField := fields[fooTypeSomeStringFieldName]
            	someString, ok = someStringField.(cadence.String)
            	if !ok {
            		return cadence.Int{}, "", fmt.Errorf("wrong field type: expected String, got %T", someString)
            	}
            	
            	return someInt, someString, nil
            }
            ```
            
        - If only a single field value is needed from the composite,
        use `cadence.SearchFieldByName`. As the name indicates, the function performs a linear search over all fields of the composite. Prefer `FieldsMappedByName` over repeated calls to `SearchFieldByName`.
        For example:
            
            ```go
            const fooTypeSomeIntFieldName = "someInt"
            
            // fooEventSomeIntField gets the value of the someInt field of the Cadence event:
            //
            //	event Foo(someInt: Int)
            //
            // It returns an error if the event does not have the expected field.
            func fooEventSomeIntField(event cadence.Event) (cadence.Int, error) {
            	someIntField := cadence.SearchFieldByName(event, fooTypeSomeIntFieldName)
            	someInt, ok := someIntField.(cadence.Int)
            	if !ok {
            		return cadence.Int{}, fmt.Errorf("wrong field type: expected Int, got %T", someInt)
            	}
            	return someInt, nil
            }
            
            ```
  </details>
            
- `cadence.GetFieldByName` got renamed to `cadence.SearchFieldByName` to make it clear that the function performs a linear search
- `cadence.GetFieldsMappedByName` got renamed to `cadence.FieldsMappedByName`, to better follow common Go style / naming guides, e.g. https://google.github.io/styleguide/go/decisions#getters
- The convenience method `ToGoValue` of `cadence.Value`, which converts a `cadence.Value` into a Go value (if possible), got removed. Likewise, the convenience function `cadence.NewValue`, which constructs a new `cadence.Value` from the given Go value (if possible), got removed.

  <details>
  <summary>Details:</summary>

    - There are many different use cases and needs for methods that convert between Cadence and Go values. When attempting to convert an arbitrary Cadence value into a Go value, there is no "correct" Go type to return in all cases. Likewise, when attempting to convert an arbitrary Go value to Cadence, there might not be a “correct” result type.
    - Developers might expect a certain Go type to be returned. For example, `ToGoValue` of `cadence.Struct` returned a Go slice, but some developers might assume and want a Go map; and `ToGoValue` of `cadence.Dictionary` returned a Go map, but did not account for the case where dictionary keys in Cadence might be types that are invalid in Go maps.
    - As the return type of `ToGoValue` is `any`, developers using the method need to cast to some expected Go type, and hope the returned value is what they expect.
    - Improvements in the implementation of `ToGoValue`, like in https://github.com/onflow/cadence/pull/2531, would have silently broken programs using the function, as the different return value would have no longer matched the developer’s expected type.
    - Even though these methods and functions got removed from the `cadence` package, developers can still perform the conversion that the `ToGoValue` methods performed. A future version of Cadence might re-introduce well-defined and strongly-typed conversion functions, that are also consistent with similar conversion functions in other languages (e.g. JavaScript SDK / FCL).
    - To see what the removed functions and methods did, have a look at the PR that removed them: https://github.com/onflow/cadence/pull/3291/files.
    - If you feel like Cadence should re-gain this functionality, please open a feature request, or even consider contributing them through pull requests

  </details>

### 🛠 Improvements
* Add runtime check for transaction value moves by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3298
* Add predicate function and test for atree's PersistentSlabStorage.FixLoadedBrokenReferences by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3300
### Other Changes
* Merge `release/v1.0.0-preview.23` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3289
* Update update tool config by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3292
* Bump atree version in master branch by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3293


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.23...v1.0.0-preview.24

[Changes][v1.0.0-preview.24]


<a name="v1.0.0-preview.23"></a>
# [v1.0.0-preview.23](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.23) - 26 Apr 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.23 -->

## What's Changed
### ⭐ Features
* Add backwards compatibility to old JSON decodings  by [@ianthpun](https://github.com/ianthpun) in https://github.com/onflow/cadence/pull/3276
### 🛠 Improvements
* Add additional fields to account key added event by [@rrrkren](https://github.com/rrrkren) in https://github.com/onflow/cadence/pull/3204
### 🐞 Bug Fixes
* Handle account types in contract update validator by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3284
* Fix restricted typed field updates by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3283
### Other Changes
* Adding staged contracts migration output JSON from Apr 17 to migratio… by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3273
* Merge `release/v1.0.0-preview.22` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3279
* Adjust branches of downstream dependency checks by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3280
* Replace colons in staged contract reports by [@jribbink](https://github.com/jribbink) in https://github.com/onflow/cadence/pull/3281
* Adding migration result for Testnet state from Apr 24 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3285
* Add basic stats and section on newly failing contracts by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3286

## New Contributors
* [@ianthpun](https://github.com/ianthpun) made their first contribution in https://github.com/onflow/cadence/pull/3276

**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.22...v1.0.0-preview.23

[Changes][v1.0.0-preview.23]


<a name="v1.0.0-preview-atree-register-inlining.23"></a>
# [v1.0.0-preview-atree-register-inlining.23](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview-atree-register-inlining.23) - 26 Apr 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview-atree-register-inlining.23 -->

## What's Changed
### ⭐ Features
* Add backwards compatibility to old JSON decodings  by [@ianthpun](https://github.com/ianthpun) in https://github.com/onflow/cadence/pull/3276
### 🛠 Improvements
* Add additional fields to account key added event by [@rrrkren](https://github.com/rrrkren) in https://github.com/onflow/cadence/pull/3204
### 🐞 Bug Fixes
* Handle account types in contract update validator by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3284
* Fix restricted typed field updates by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3283
### Other Changes
* Adding staged contracts migration output JSON from Apr 17 to migratio… by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3273
* Merge `release/v1.0.0-preview.22` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3279
* Adjust branches of downstream dependency checks by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3280
* Replace colons in staged contract reports by [@jribbink](https://github.com/jribbink) in https://github.com/onflow/cadence/pull/3281
* Adding migration result for Testnet state from Apr 24 by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3285
* Add basic stats and section on newly failing contracts by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3286

## New Contributors
* [@ianthpun](https://github.com/ianthpun) made their first contribution in https://github.com/onflow/cadence/pull/3276

**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.22...v1.0.0-preview-atree-register-inlining.23

[Changes][v1.0.0-preview-atree-register-inlining.23]


<a name="v1.0.0-preview.22"></a>
# [v1.0.0-preview.22](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.22) - 24 Apr 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.22 -->

## What's Changed
### 💥 Language Breaking Changes
* Implement FLIP 242 by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3252
* Implement FLIP 262 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3247
* Fix optional type ID ambiguity and migrate types by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3272
* Improve supported entitlements and entitlements inferrence in migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3266
### 🛠 Improvements
* Improve conformance mismatch error by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3274
* Further improve conformance error by including location by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3277
* Remove unused parameter by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3278
* Cache results of entitlement type conversion by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3232
### Other Changes
* Merge `release/v1.0.0-preview.21` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3269
* Adding migration result for staged contracts migration, testnet, Apri… by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3263


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.21...v1.0.0-preview.22

[Changes][v1.0.0-preview.22]


<a name="v1.0.0-preview-atree-register-inlining.22"></a>
# [v1.0.0-preview-atree-register-inlining.22](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview-atree-register-inlining.22) - 24 Apr 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview-atree-register-inlining.22 -->

## What's Changed
### 💥 Language Breaking Changes
* Implement FLIP 242 by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3252
* Implement FLIP 262 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3247
* Fix optional type ID ambiguity and migrate types by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3272
* Improve supported entitlements and entitlements inferrence in migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3266
### 🛠 Improvements
* Improve conformance mismatch error by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3274
* Further improve conformance error by including location by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3277
* Remove unused parameter by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3278
* Cache results of entitlement type conversion by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3232
### Other Changes
* Adding migration result for staged contracts migration, testnet, Apri… by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3263


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.21...v1.0.0-preview-atree-register-inlining.22

[Changes][v1.0.0-preview-atree-register-inlining.22]


<a name="v1.0.0-preview.21"></a>
# [v1.0.0-preview.21](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.21) - 18 Apr 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.21 -->

## What's Changed
### 🛠 Improvements
* Improve dictionary key conflict handling by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3240
* Improve supported entitlements by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3251
### 🐞 Bug Fixes
* Introduce special self-variable by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3235
* Fix type indexing resource loss by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3257
* Fix resource-loss check in dictionary set-key by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3267
### 🧪 Testing
* Add a test case for non-public interface members by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3260
### Other Changes
* Merge `release/v1.0.0-preview.20` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3239
* Tool for converting staged contracts migration report from JSON to Markdown by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3249
* Bump atree version in master branch  by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3254
* Fix staged contracts report printer - update JSON field names by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/3262


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.20...v1.0.0-preview.21

[Changes][v1.0.0-preview.21]


<a name="v1.0.0-preview.20"></a>
# [v1.0.0-preview.20](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.20) - 12 Apr 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.20 -->

## What's Changed
### 💥 Language Breaking Changes
* Remove ability to dereference arrays or dicts of structs by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3221
### 🛠 Improvements
* Check resource loss in `CompositeValue.RemoveField` by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3224
* Check resource loss in all variable assignments by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3226
* Introduce a dedicated error for unreferenced slabs and expose the IDs by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3231
* Move conflicting dictionary key to new dictionary in unique storage path by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3227
### 🐞 Bug Fixes
* Handle missing type arguments in type argument checks by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3234
* Clear `inAssignment` when recursing into subexpressions by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3236
### Other Changes
* Merge `release/v1.0.0-preview.19` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3217


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.19...v1.0.0-preview.20

[Changes][v1.0.0-preview.20]


<a name="v0.42.10"></a>
# [v0.42.10](https://github.com/onflow/cadence/releases/tag/v0.42.10) - 10 Apr 2024

<!-- Release notes generated using configuration in .github/release.yml at v0.42.10 -->

## What's Changed
### 🛠 Improvements
* Port [#3231](https://github.com/onflow/cadence/issues/3231) to v0.42 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3237
### 🐞 Bug Fixes
* Handle optional storage references by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3094


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.42.9...v0.42.10

[Changes][v0.42.10]


<a name="v1.0.0-preview.19"></a>
# [v1.0.0-preview.19](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.19) - 03 Apr 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.19 -->

## What's Changed

### 🛠 Improvements
* Improve Cadence 1.0 state migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3211

### 🐞 Bug Fixes
* Properly handle `Never` type argument to `Capabilities.get` by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3203
* Add `TypeArgumentsCheck`s to certain builtin functions by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3208
* Proper type checking for resource use after validation on two-value transfers by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3176
* Convert references based on borrow type by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3213
* Fix migration of built-in types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3205
* Resource Reference Invalidation Improvements by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3181
* Fix built-in type import by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3212
* Fix dereferencing non-reference type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3216
* Port fixes of v0.42.8-patch.4 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3079
 
### Other Changes
* Merge `release/v1.0.0-preview.18` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3202



**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.18...v1.0.0-preview.19

[Changes][v1.0.0-preview.19]


<a name="v1.0.0-preview.18"></a>
# [v1.0.0-preview.18](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.18) - 27 Mar 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.18 -->

## What's Changed
### 🛠 Improvements
* Simplify AuthorizationMismatchError by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3201


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.17...v1.0.0-preview.18

[Changes][v1.0.0-preview.18]


<a name="v1.0.0-preview.17"></a>
# [v1.0.0-preview.17](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.17) - 27 Mar 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.17 -->

## What's Changed
### ⭐ Features
* Storage Explorer by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3147
### 🛠 Improvements
* Make the use of custom contract update rules mandatory, if exists by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3193
* Allow removal of enum declarations from contract interfaces for C1.0 upgrade by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3197
* Name storage migration, include in error by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3198
### 🐞 Bug Fixes
* Handle unparameterized Capability static types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3196
* Fix address conversion by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3195
* Only encode reference static type in legacy format if it was decoded as such by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3199
* Fix conformance checking for enums in contract update validator by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3191
### 📖 Documentation
* Improve development documentation by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3189
### Other Changes
* Merge `release/v1.0.0-preview.16` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3185
* Update CODEOWNERS by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3190


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.16...v1.0.0-preview.17

[Changes][v1.0.0-preview.17]


<a name="v1.0.0-preview.16"></a>
# [v1.0.0-preview.16](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.16) - 20 Mar 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.16 -->

## What's Changed
### 🛠 Improvements
* Relax interface conformance changes in contract update validator by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3184
### Other Changes
* Merge `release/v1.0.0-preview.15` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3180


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.15...v1.0.0-preview.16

[Changes][v1.0.0-preview.16]


<a name="v1.0.0-preview.15"></a>
# [v1.0.0-preview.15](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.15) - 18 Mar 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.15 -->

## What's Changed
### 🛠 Improvements
* Improve update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3175
### Other Changes
* Merge `release/v1.0.0-preview.14` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3174


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.14...v1.0.0-preview.15

[Changes][v1.0.0-preview.15]


<a name="v1.0.0-preview.14"></a>
# [v1.0.0-preview.14](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.14) - 13 Mar 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.14 -->

## What's Changed
### 🛠 Improvements
* Meter computation for standard-library functions by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3172
* Optimize string migration: only return new value if needed by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3173
### Other Changes
* Merge `release/v1.0.0-preview.13` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3171


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.13...v1.0.0-preview.14

[Changes][v1.0.0-preview.14]


<a name="v1.0.0-preview.13"></a>
# [v1.0.0-preview.13](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.13) - 13 Mar 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.13 -->

## What's Changed
### 💥 Language Breaking Changes
* Allow creating references to nested optionals by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3132
### 🛠 Improvements
* Improve update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3163
* Improve migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3169
### 🐞 Bug Fixes
* Implicitly track composite reference in attachment iteration by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3168
### Other Changes
* Merge `release/v1.0.0-preview.12` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3167


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.12...v1.0.0-preview.13

[Changes][v1.0.0-preview.13]


<a name="v1.0.0-preview.12"></a>
# [v1.0.0-preview.12](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.12) - 11 Mar 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.12 -->

## What's Changed
### 🐞 Bug Fixes
* Fix intersection type's legacy type getting converted to intersection type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3166
### Other Changes
* Merge `release/v1.0.0-preview.11` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3165


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.11...v1.0.0-preview.12

[Changes][v1.0.0-preview.12]


<a name="v1.0.0-preview.11"></a>
# [v1.0.0-preview.11](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.11) - 11 Mar 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.11 -->

## What's Changed
### 🛠 Improvements
* Handle legacy type getting converted to intersection type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3164
### 🐞 Bug Fixes
* Check invalidation of the looped reference by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3160
### Other Changes
* Update version by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3161


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.10...v1.0.0-preview.11

[Changes][v1.0.0-preview.11]


<a name="v1.0.0-preview.10"></a>
# [v1.0.0-preview.10](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.10) - 08 Mar 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.10 -->

## What's Changed
### 🛠 Improvements
* Improve migrations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3144
* Improve errors in state migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3154
* Improve type inference for authorized references by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3156
* Improve static type and entitlements migrations: Update array/dictionary type directly by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3158
* Optimize storage migration: Allow skipping of values by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3157
### 🐞 Bug Fixes
* Fix location ranges by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3151
* Produce a better error message when failing to reference type-erased references by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3150
* Make SupportedEntitlements thread safe by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3152

**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-M9...v1.0.0-preview.10

[Changes][v1.0.0-preview.10]


<a name="v1.0.0-M9"></a>
# [v1.0.0-M9](https://github.com/onflow/cadence/releases/tag/v1.0.0-M9) - 04 Mar 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-M9 -->

## What's Changed

### 💥 Language Breaking Changes
* Prevent container mutation while iterating by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3126

### ⭐ Features
* Add support for `FunctionType.Purity` in CCF codec by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3107
* Add support for Attachment and AttachmentType in CCF codec by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3131
* Add more support for entitlements (`ReferenceType.Authorization`) to CCF codec by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3139

### 🛠 Improvements
* Improve update tool: Fix config, improve documentation by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3130
* Require added entitlements to be equal to migrated entitlements rather than supertypes by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3134
* Fix migrating values with empty intersection type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3138
* Pass location-range to value-interface methods by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3140

### 🐞 Bug Fixes
* Fix EntitlementSetAuthorization.Equal() by making it ignore order of elements by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3137
* Migrate static types of arrays and dictionaries by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3141

### 🧪 Testing
* Add test for nested container value migration by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3142


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-M8...v1.0.0-M9

[Changes][v1.0.0-M9]


<a name="v1.0.0-M8"></a>
# [v1.0.0-M8](https://github.com/onflow/cadence/releases/tag/v1.0.0-M8) - 22 Feb 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-M8 -->

## What's Changed
### ⭐ Features
* Contract Update Validator by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3069
### 🛠 Improvements
* Improve migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3110
* Improve update tool: Allow releasing a particular branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3124
* Improve Cadence-1.0 contract update checker by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3111
* Fix location collection in contract update validator by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3117
* Convert untyped path capability values to ID capability values by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3125
* Allow registering custom type updating rules in contract update validator by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3120
* Rewrite owned intersection types in migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3118
### 🧪 Testing
* Add tests for migrating reference to optional type by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3123
### Other Changes
* Merge `release/v1.0.0-M7` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3106
* Dependencies got updated to onflow/flowkit/v2 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3109


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-M7...v1.0.0-M8

[Changes][v1.0.0-M8]


<a name="v1.0.0-M7"></a>
# [v1.0.0-M7](https://github.com/onflow/cadence/releases/tag/v1.0.0-M7) - 14 Feb 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-M7 -->

## What's Changed
### 💥 Language Breaking Changes
* Unify the requirement for all interface types to have to occur in intersection types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3090
### 🛠 Improvements
* Add more information to analysis.Diagnostic by [@jribbink](https://github.com/jribbink) in https://github.com/onflow/cadence/pull/3082
* Report a proper error for swapping a resource to itself  by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3101
* Bring back support for decoding published path capabilities by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3103
* Allow analysis.Load to continue with parsing/checking errors by [@jribbink](https://github.com/jribbink) in https://github.com/onflow/cadence/pull/3083
### 🐞 Bug Fixes
* Consider inherited interfaces when checking for member clashes in intersection types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3093
* Fix FunctionType.Equal() to check Purity equality by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/3105
### Other Changes
* Merge `release/v1.0.0-M6` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3100
* Update configuration of downstream dependencies by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3095


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-M6...v1.0.0-M7

[Changes][v1.0.0-M7]


<a name="v1.0.0-M6"></a>
# [v1.0.0-M6](https://github.com/onflow/cadence/releases/tag/v1.0.0-M6) - 12 Feb 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-M6 -->

## What's Changed
### 🛠 Improvements
* Improve migrations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3072
* Improve update script by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3070
* Add support for migrating path capability values in entitlements and static type migrations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3073
* Replace existing tool to fetch network contracts with Flowdiver-based one by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3058
* Improve member resolvers by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3092
### 🐞 Bug Fixes
* Remove reference to `destroy` from sema.UnknownSpecialFunctionError by [@jribbink](https://github.com/jribbink) in https://github.com/onflow/cadence/pull/3074
* Port Validator fix to master by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3078
* Fix intersection type migration: also migrate interface types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3084
* Make capability ID mapping thread-safe by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3085
* Allow contract interface types in intersection types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3089
### 📖 Documentation
* Fixes links to new docs site by [@joshuahannan](https://github.com/joshuahannan) in https://github.com/onflow/cadence/pull/3075
### Other Changes
* Merge `release/v1.0.0-M5` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3067


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-M5...v1.0.0-M6

[Changes][v1.0.0-M6]


<a name="v0.42.9"></a>
# [v0.42.9](https://github.com/onflow/cadence/releases/tag/v0.42.9) - 07 Feb 2024

<!-- Release notes generated using configuration in .github/release.yml at v0.42.9 -->

## What's Changed

### 🐞 Bug Fixes

* Port fixes of v0.42.8-patch.4 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3076

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.42.8...v0.42.9

[Changes][v0.42.9]


<a name="v1.0.0-M5"></a>
# [v1.0.0-M5](https://github.com/onflow/cadence/releases/tag/v1.0.0-M5) - 02 Feb 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-M5 -->

## What's Changed
### ⭐ Features
* Allow reading from CSV and querying AST using (Go) JQ by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3051
### 🛠 Improvements
* Add Unwrap method for ParentErrors by [@peterargue](https://github.com/peterargue) in https://github.com/onflow/cadence/pull/3052
* Don't skip invalid entitlements when creating errors by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3056
* Gracefully handle runtime panics in migration by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3066
### 🐞 Bug Fixes
* Handle optional storage references by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3064
* Fix entitlements migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3065
### 🧪 Testing
* Add tests for using keywords as field names by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3057
* Minor improvement in a `revertibleRandom` test by [@tarakby](https://github.com/tarakby) in https://github.com/onflow/cadence/pull/3060

**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-M4...v1.0.0-M5

[Changes][v1.0.0-M5]


<a name="v1.0.0-M4"></a>
# [v1.0.0-M4](https://github.com/onflow/cadence/releases/tag/v1.0.0-M4) - 26 Jan 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-M4 -->

## What's Changed
### 💥 Go API Breaking Chance
* Expose checker in analysis by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3042
### ⭐ Features
* Use old parser for contract upgrades when legacy upgrade config option is set by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3043
### 🐞 Bug Fixes
* Remove backticks around suggested fixes for parser errors by [@jribbink](https://github.com/jribbink) in https://github.com/onflow/cadence/pull/3046
### 🧪 Testing
* Test usage of potentially broken types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3030
### Other Changes
* Merge `release/v1.0.0-M3` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3041

## New Contributors
* [@jribbink](https://github.com/jribbink) made their first contribution in https://github.com/onflow/cadence/pull/3046

**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-M3...v1.0.0-M4

[Changes][v1.0.0-M4]


<a name="v0.42.8"></a>
# [v0.42.8](https://github.com/onflow/cadence/releases/tag/v0.42.8) - 26 Jan 2024

<!-- Release notes generated using configuration in .github/release.yml at v0.42.8 -->

## What's Changed

### ⭐ Features
* Add getter for field count to composite value by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3003
* Allow parsing code in CSV and querying using (Go) JQ by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3022

### 🛠 Improvements
* Add Unwrap method for ParentErrors - v0.42 backport by [@peterargue](https://github.com/peterargue) in https://github.com/onflow/cadence/pull/3053

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.42.7...v0.42.8

[Changes][v0.42.8]


<a name="v1.0.0-M3"></a>
# [v1.0.0-M3](https://github.com/onflow/cadence/releases/tag/v1.0.0-M3) - 24 Jan 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-M3 -->

## What's Changed
### 💥 Language Breaking Changes
* Reserve more keywords for future proofing by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3019
* Require explicit type annotation for arguments passed to authorized reference parameters by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3023
### ⭐ Features
* Introduce `toConstantSized` to `VariableSizedArray` type by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/3028
* Add v0.42 parser package under old_parser by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3039
### 🛠 Improvements
* Improve parsing for backward compatibility rules by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3021
* Fix the entitlements migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3026
* Use legacy hashing function for key removal by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3013
* Allow dereferencing references to containers of non-resources by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3034
* Generalize static type migrations by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3033
### 🐞 Bug Fixes
* Fix `revertibleRandom` with module test by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3025
* Remove Insert-entitled access for toVariableSized by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3031
* Fix gaps in migrations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3036
### Other Changes
* Merge `release/v1.0.0-preview.2` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/3024
* Upgrade `exp` package version by [@tarakby](https://github.com/tarakby) in https://github.com/onflow/cadence/pull/3027


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.2...v1.0.0-M3

[Changes][v1.0.0-M3]


<a name="v1.0.0-preview.2"></a>
# [v1.0.0-preview.2](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.2) - 16 Jan 2024

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.2 -->

## What's Changed
### 💥 Language Breaking Changes
* Eagerly normalize String and Character values at construction time by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2781
* Add required `mapping` keyword to entitlement mapping usages by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2883
* Restrict `capabilities.publish` to account's own capabilities by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2782
* Remove support for custom destructors by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2789
* Support for parsing default destroy events by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2799
* Type checking for default events by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2812
* Merge destructor removal into stable cadence by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2928
* throw error on the creation of a nested reference by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2965
* New Behavior for Attachments and Entitlements by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2921
* Merge Stable Cadence branch into master by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2971
* Update interface of `revertibleRandom` by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2976
* Static check to prevent nested references by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2970
* Remove deprecated domain separation tag from Crypto contract by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2984
* Implement modulo argument of `revertibleRandom` by [@tarakby](https://github.com/tarakby) in https://github.com/onflow/cadence/pull/2986
### 💥 Go API Breaking Changes
* Add support for injecting types into the environment by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2811
* Allow different base activations per location in checker and interpreter by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2887
### ⭐ Features
* Add `tryUpdate` method to `Account.Contracts` by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2769
* Add an exists function, allows checking if capability exists/is published by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2778
* Introduce `String.join` function by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2762
* Introduce `String.split` function by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2791
* Check native function declarations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2821
* Allow native functions to have type parameters by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2850
* Introduce `String.replaceAll` instance function by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2814
* Introduce `InclusiveRange<T: Integer>` type by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2523
* Allow specifying a mapping with source paths for locations in coverage reports by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2859
* Add support iterating references to iterables by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2876
* Allow injection of functions into composite values, refactor PublicKey based on it by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2878
* add new `revertibleRandom` function by [@tarakby](https://github.com/tarakby) in https://github.com/onflow/cadence/pull/2896
* Introduce `HashableStruct` type to represent Dictionary keys by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2872
* Allow `InclusiveRange<T>` in a for-in loop by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2892
* Add account type migration by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2923
* Add String normalizing migration by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2937
* Concretize the definition of Primitive type by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2975
* Capability controllers migration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2912
* Add state migration for removing all values from private domain by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2974
* State migration for type values with two or more interfaces by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2981
* Add getter for field count to composite value by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3002
* Introduce dereference operator for references by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2982
* Add support for dereferencing optional references by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3008
* Merge range type feature by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3009
* Introduce `toVariableSized` to ConstantSizedArray type by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/3016
* Allow issuing capability controllers using type values by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3018
* Entitlements Migration by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2951
### 🛠 Improvements
* Replace `simpleTypeIDByType` with a bimap by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2775
* Use bimaps in the elaboration by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2779
* Refactor runtime test helpers into separate, reusable package by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2800
* Add helpers to get constructor and initializer function type for composite type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2805
* Replace panic with nil return in `newInjectedCompositeFieldsHandler` by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2808
* Extend type code generator by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2806
* Refine API of the testing framework by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2783
* mjs build for cadence-parser by [@nialexsan](https://github.com/nialexsan) in https://github.com/onflow/cadence/pull/2851
* Improve test runtime interface by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2894
* Allow default functions to co-exist with empty function declarations by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2725
* Refactor resource-reference tracking by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2916
* Add types to parsed array and dictionary literals by [@bluesign](https://github.com/bluesign) in https://github.com/onflow/cadence/pull/2924
* AST improvements by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2949
* Disallow InclusiveRange<T> if T is a non-leaf integer by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2959
* Improve resource-reference tracking by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2958
* Generalize the migrations and make common codes re-usable by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2942
* Improve Account type migration by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2973
* Make entitlement set access incomparable by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2972
* Use IsPrimitiveType to check default destructor params by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2980
* Various improvements by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2979
* Report bespoke error for restricted types parsed as type arguments by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2985
* Use old hash/ID generation to remove keys by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2987
* Extend pragma expressions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2993
* Improve error handling in migrations by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/3001
* Improve type inference by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3004
* Add type assertion on values passed to `Test.assertEqual` by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/3014
* Improve code related to new InclusiveRange type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3017
* Defensively check the dereferenced value is not a resource by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/3010
### 🐞 Bug Fixes
* Fix doc (pretty printing) for function declaration without function block by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2774
* Port internal [#143](https://github.com/onflow/cadence/issues/143) by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2793
* Properly check removed expression for resource loss by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2798
* Fix Test framework's TestAccount's type name by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2802
* Fix capability controller deletion by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2788
* meter and deduplicate included entitlement relations by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2810
* Use kr/pretty instead of go-test/deep to prevent empty diffs, remove gocap by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2819
* properly access-check optional chaining with entitlements by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2825
* Check `before` statements in `view` contexts by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2835
* Skip adding malformed entitlement relations to maps by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2838
* Require full codomain of map when assigning to mapped field by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2840
* Port internal 141: Fix swapping in resource array by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2844
* Port internal 144: Fix swap statement evaluation by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2845
* Port internal fix 145: Prevent re-deploy in same transaction by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2846
* Properly set base on loaded attachments during attachment iteration by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2847
* Fix memory metering for loading stored values by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2509
* [stable-cadence] Fix field assignment via references by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2868
* Entitlement mapping escape fixes by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2877
* Fix string atree value comparison: handle storage as slab by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2895
* Removing an attachment is an impure operation by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2890
* [v0.42] Prevent nested contract deployments by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2904
* Prevent nested contract deployments by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2902
* [v0.42] Fix nested resource moves by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2930
* [v0.42] Fix AuthAccount creation by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2932
* Fix nested resource moves by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2931
* Fix AuthAccount creation by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2933
* Fix transaction resource-typed field move by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2938
* Include interface conformances when computing the supported entitlements of an interface  by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2946
* Port bug fixes from `v0.42.5-patch.1` by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2955
* Port bug fixes from v0.42.5-patch.1 by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2956
* Fix GetNativeCompositeValueComputedFields by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2977
* Statically prevent referenced-typed subexpressions in default arguments to destroy events by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2996
* Prevent use of `base` variable in default destroy event arguments outside attachments by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2999
* Prevent use of non-attachment type names in default arguments by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3000
* prevent creation of nested storage references by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/3020
### 🧪 Testing
* Add test for container mutation while iterating by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2960
* Test type order insignificance by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2967
### 📖 Documentation
* Add notes for 2023-09-12 meeting by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2780
* Add meeting notes for 2023-11-15 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2944
### Other Changes
* Sync `feature/range-type` branch with `master` by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2747
* Merge `release/v0.41.0` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2786
* Merge `release/v0.41.1` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2795
* Sync Stable Cadence by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2796
* Assign issues to group by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2797
* Remove unused composite type to interface type conversion function by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2804
* Sync Stable Cadence  by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2820
* Merge `release/v0.42.0` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2852
* Sync Stable Cadence by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2854
* Add computation metering to the new stdlib functions by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2880
* Use `ComputationKindLoop` for internal array-value iterations by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2891
* Merge `release/v0.42.1` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2893
* Add meeting notes for 2023-10-24 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2897
* Sync stable cadence branch with master by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2899
* Merge `release/v0.42.2` to `v0.42` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2905
* [v0.42] Port adding new `revertibleRandom` function by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2910
* Merge `release/v0.42.3` to `v0.42` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2911
* Update tests for `Crypto` contract to the latest testing framework API by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2856
* Remove `StandardLibraryHandler` method from `TestFramework` interface by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2927
* Merge `release/v0.42.4` to `v0.42` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2934
* Fix Semgrep by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2939
* Merge v0.42 into master by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2940
* Merge `release/v0.42.5` to `v0.42` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2941
* Sync stable-cadence branch with master by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2964
* Remove unsafeRandom by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2966
* Sync stable-cadence branch with master by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2968
* Cleanup reference tracking by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2969
* Make checkParameterizedTypeIsInstantiated recursive by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2915
* Sync `feature/range-type` branch with `master` by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2995


**Full Changelog**: https://github.com/onflow/cadence/compare/v1.0.0-preview.1...v1.0.0-preview.2

[Changes][v1.0.0-preview.2]


<a name="v0.42.7"></a>
# [v0.42.7](https://github.com/onflow/cadence/releases/tag/v0.42.7) - 05 Jan 2024

<!-- Release notes generated using configuration in .github/release.yml at v0.42.7 -->

## What's Changed
### ⭐ Features
* Port `tryUpdate` to v0.42 by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2991

[Changes][v0.42.7]


<a name="v0.42.6"></a>
# [v0.42.6](https://github.com/onflow/cadence/releases/tag/v0.42.6) - 30 Nov 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.42.6 -->

## What's Changed
### 🐞 Bug Fixes
* Port bug fixes from `v0.42.5-patch.1` by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2955
### Other Changes
* Merge `release/v0.42.5` to `v0.42` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2941


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.42.5...v0.42.6

[Changes][v0.42.6]


<a name="v0.42.5"></a>
# [v0.42.5](https://github.com/onflow/cadence/releases/tag/v0.42.5) - 09 Nov 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.42.5 -->

## What's Changed
### 🐞 Bug Fixes
* Fix transaction resource-typed field move by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2938


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.42.4...v0.42.5

[Changes][v0.42.5]


<a name="v0.42.4"></a>
# [v0.42.4](https://github.com/onflow/cadence/releases/tag/v0.42.4) - 09 Nov 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.42.4 -->

## What's Changed
### 🐞 Bug Fixes
* Fix nested resource moves by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2930
* Fix AuthAccount creation by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2932
### Other Changes
* Merge `release/v0.42.3` to `v0.42` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2911


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.42.3...v0.42.4

[Changes][v0.42.4]


<a name="v0.42.3"></a>
# [v0.42.3](https://github.com/onflow/cadence/releases/tag/v0.42.3) - 30 Oct 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.42.3 -->

## What's Changed
### ⭐ Features
* Port adding new `revertibleRandom` function by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2910
### Other Changes
* Merge `release/v0.42.2` to `v0.42` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2905

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.42.2...v0.42.3

[Changes][v0.42.3]


<a name="v0.42.2"></a>
# [v0.42.2](https://github.com/onflow/cadence/releases/tag/v0.42.2) - 26 Oct 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.42.2 -->

## What's Changed
### 🐞 Bug Fixes
* [v0.42] Prevent nested contract deployments by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2904


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.42.1...v0.42.2

[Changes][v0.42.2]


<a name="v0.39.17"></a>
# [v0.39.17](https://github.com/onflow/cadence/releases/tag/v0.39.17) - 26 Oct 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.17 -->

## What's Changed
### 🐞 Bug Fixes
* [v0.39] Prevent nested contract deployments by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2901
### Other Changes
* Merge `release/v0.39.16` to `v0.39` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2849


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.16...v0.39.17

[Changes][v0.39.17]


<a name="v0.42.1"></a>
# [v0.42.1](https://github.com/onflow/cadence/releases/tag/v0.42.1) - 23 Oct 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.42.1 -->

## What's Changed

### ⭐ Features
* Allow native functions to have type parameters by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2850
* Allow specifying a mapping with source paths for locations in coverage reports by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2859
### 🛠 Improvements
* mjs build for cadence-parser by [@nialexsan](https://github.com/nialexsan) in https://github.com/onflow/cadence/pull/2851
### 💥 Go API Breaking Chance
* Allow different base activations per location in checker and interpreter by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2887
### Other Changes
* Add computation metering to the new stdlib functions by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2880
* Use `ComputationKindLoop` for internal array-value iterations by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2891


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.42.0...v0.42.1

[Changes][v0.42.1]


<a name="v0.42.0"></a>
# [v0.42.0](https://github.com/onflow/cadence/releases/tag/v0.42.0) - 05 Oct 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.42.0 -->

## What's Changed

### ⭐ Features
* Check native function declarations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2821

### 🛠 Improvements

* Replace panic with nil return in `newInjectedCompositeFieldsHandler` by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2808
* Refine API of the testing framework by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2783

### 🐞 Bug Fixes
* Properly check removed expression for resource loss by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2798
* Fix capability controller deletion by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2788
* Use kr/pretty instead of go-test/deep to prevent empty diffs, remove gocap by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2819
* Fix swapping in resource array by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2844
* Fix swap statement evaluation by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2845
* Prevent re-deploy in same transaction by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2846

### 💥 Go API Breaking Chance
* Add support for injecting types into the environment by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2811

### Other Changes
* Merge `release/v0.41.1` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2795
* Assign issues to group by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2797


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.41.1...v0.42.0

[Changes][v0.42.0]


<a name="v0.39.16"></a>
# [v0.39.16](https://github.com/onflow/cadence/releases/tag/v0.39.16) - 05 Oct 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.16 -->

## What's Changed

### 🐞 Bug Fixes
* Fix swapping in resource array by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2841
* Fix swap statement evaluation by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2842
* Prevent re-deploy in same transaction by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2843

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.15...v0.39.16

[Changes][v0.39.16]


<a name="v1.0.0-preview.1"></a>
# [v1.0.0-preview.1](https://github.com/onflow/cadence/releases/tag/v1.0.0-preview.1) - 21 Sep 2023

<!-- Release notes generated using configuration in .github/release.yml at v1.0.0-preview.1 -->

## What's Changed
### 💥 Language Breaking Changes
* Change member access semantics by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2573
* Add builtin entitlements to array/dictionary functions by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2598
* Allow external mutation for owned values by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2599
* Make indexing assignment entitled by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2664
* Allow default functions to co-exist with conditions in interface conformance by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2697
* Remove linking based capability API and associated code by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2595
* Add external mutability improvements  by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2718
* Remove type requirements by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2670
* sort entitlement sets alphabetically when generating strings by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2754
* Account type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2648
* Improve and fix static types and their ID and string functions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2756
* Add view annotations to newly added array functions by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2771
### 💥 Go API Breaking Chance
* Update randomness runtime interface method by [@tarakby](https://github.com/tarakby) in https://github.com/onflow/cadence/pull/2706
* Add the `NewEmulatorBackend` method on `TestFramework` interface by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2744
### ⭐ Features
* Introduce built-in mutability entitlements by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2586
* Allow entitlement mappings to be used in container-typed composite-fields by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2587
* Add identity entitlement mapping by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2588
* Return a reference with all entitlements, when access through owned values by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2637
* Introduce `filter` in Fixed/Variable sized Array types by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2678
* Allow declaration of events in interfaces by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2662
* Add a utility method to the `Test.Blockchain` struct for moving the blockchain time by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2729
* Add capability field to capabilitity controllers by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2724
* Introduce `map` in Fixed/Variable sized Array types by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2737
* Composition of entitlement mappings by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2743
* Allow use of type code generator in any package, refactor RLP and BLS contracts by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2753
* Add functions on testing framework's blockchain to create/load snapshots by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2763
### 🛠 Improvements
* Temporarily remove deprecations for existing capability API by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2673
* Cleanup by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2692
* Improve view annotations for account type functions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2699
* Cadence testing framework improvements by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2696
* Clean up capability value by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2727
* Add purity/view annotations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2728
* Entitlements error improvements by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2730
* Allow `moveTime` function to accept negative time deltas for going backwards in time by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2734
* Give `CompositeTypeHandler` higher precedence over `getUserCompositeType` by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2717
* Improve REPL by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2731
* add suggestions for missing entitlements in access errors by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2736
* Improve error message for overloaded declaration by [@PratikDhanave](https://github.com/PratikDhanave) in https://github.com/onflow/cadence/pull/2738
* Improve composite and interface static types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2751
* Avoid unnecessary static to sema type conversions for `Type`  by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2750
* Make CodesAndPrograms public by [@janezpodhostnik](https://github.com/janezpodhostnik) in https://github.com/onflow/cadence/pull/2746
* update account type mappings to include identity by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2761
* Add entitlement CopyValue and require it for Account.Storage.copyValue by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2765
* Better error messages for use of old restricted types by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2764
* Suggested fixes for `pub` and `priv` parser errors by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2767
* Improve capability API by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2772
### 🐞 Bug Fixes
* Wrap host env errors with external errors by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2683
* Fix interface member inheritance by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2686
* Fix storage iteration and error misclassification by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2676
* Fix error on reference creation with invalid type by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2687
* Invalidate references to nested resources upon the move of the outer resource by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2460
* Hex encode user input in error to avoid invalid UTF-8 by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2704
* Use thread safe data structures for entitlement maps by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2716
* Properly meter the creation of entitlement static types by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2723
### 📖 Documentation
* Add meeting notes for 2023-08-15 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2740
### Other Changes
* Sync mutability restrictions branch with stable-cadence by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2600
* Rename mutability entitlements by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2690
* Update to Go 1.20, improve random value generation by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2694
* Merge `release/v0.40.0` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2695
* Update storage iteration value check and related tests by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2702
* Sync Stable Cadence branch with master by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2701
* Update mutability restrictions feature branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2714
* Add test for borrowing as inherited interface by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2722
* Add test for attachment on built-in composite by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2732
* Add test for inner container mutation while iterating outer by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2733
* Add `cadence-tools/test` as a dependency to the language server by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2742
* Sync Stable Cadence feature branch by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2755
* Clean up intersection types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2757
* Add tests for invalidation of references created with index/field-access by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2758
* Merge `master` into Stable Cadence by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2770

## New Contributors
* [@PratikDhanave](https://github.com/PratikDhanave) made their first contribution in https://github.com/onflow/cadence/pull/2738

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.41.0-stable-cadence.1...v1.0.0-preview.1

[Changes][v1.0.0-preview.1]


<a name="v0.41.1"></a>
# [v0.41.1](https://github.com/onflow/cadence/releases/tag/v0.41.1) - 19 Sep 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.41.1 -->

## What's Changed
### ⭐ Features
* Introduce `String.join` function by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2762
### 🐞 Bug Fixes
* Port internal [#143](https://github.com/onflow/cadence/issues/143) by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2793

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.41.0...v0.41.1

[Changes][v0.41.1]


<a name="v0.39.15"></a>
# [v0.39.15](https://github.com/onflow/cadence/releases/tag/v0.39.15) - 19 Sep 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.15 -->

## What's Changed
### 🐞 Bug Fixes
* Port internal [#143](https://github.com/onflow/cadence/issues/143) by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2792

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.14...v0.39.15

[Changes][v0.39.15]


<a name="v0.41.0"></a>
# [v0.41.0](https://github.com/onflow/cadence/releases/tag/v0.41.0) - 14 Sep 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.41.0 -->

## What's Changed
### 💥 Go API Breaking Chance
* Update randomness runtime interface method by [@tarakby](https://github.com/tarakby) in https://github.com/onflow/cadence/pull/2706
* Add the `NewEmulatorBackend` method on `TestFramework` interface by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2744
### ⭐ Features
* Introduce `filter` in Fixed/Variable sized Array types by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2678
* Add a utility method to the `Test.Blockchain` struct for moving the blockchain time by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2729
* Add capability field to capabilitity controllers by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2724
* Introduce `map` in Fixed/Variable sized Array types by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2737
* Add functions on testing framework's blockchain to create/load snapshots by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2763
### 🛠 Improvements
* Cadence testing framework improvements by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2696
* Allow `moveTime` function to accept negative time deltas for going backwards in time by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2734
* Give `CompositeTypeHandler` higher precedence over `getUserCompositeType` by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2717
* Improve error message for overloaded declaration by [@PratikDhanave](https://github.com/PratikDhanave) in https://github.com/onflow/cadence/pull/2738
* Improve composite and interface static types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2751
* Avoid unnecessary static to sema type conversions for `Type`  by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2750
* Make CodesAndPrograms public by [@janezpodhostnik](https://github.com/janezpodhostnik) in https://github.com/onflow/cadence/pull/2746
### 🐞 Bug Fixes
* Hex encode user input in error to avoid invalid UTF-8 by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2704
* Fix doc (pretty printing) for function declaration without function block by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2774
### 🧪 Testing
* Add test for attachment on built-in composite by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2732
* Add test for inner container mutation while iterating outer by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2733
### 📖 Documentation
* Add meeting notes for 2023-08-15 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2740
* Add notes for 2023-09-12 meeting by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2780
### Other Changes
* Update to Go 1.20, improve random value generation by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2694
* Merge `release/v0.40.0` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2695
* Add `cadence-tools/test` as a dependency to the language server by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2742

## New Contributors
* [@PratikDhanave](https://github.com/PratikDhanave) made their first contribution in https://github.com/onflow/cadence/pull/2738

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.40.0...v0.41.0

[Changes][v0.41.0]


<a name="v0.41.0-stable-cadence.1"></a>
# [v0.41.0-stable-cadence.1](https://github.com/onflow/cadence/releases/tag/v0.41.0-stable-cadence.1) - 29 Aug 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.41.0-stable-cadence.1 -->

## What's Changed
### 💥 Language Breaking Changes
* Entitlements
    * Parsing for Entitlements by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2363
    * Type checking for entitlement and entitlement mapping declarations by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2441
    * Type checking for authorized references by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2405
    * Runtime behavior for entitlements by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2421
    * Type checking for entitlement and entitlement mapping declarations by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2404
* Reference invalidation
    * Invalidate resource references when the referenced resource is moved by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1999
    * Detect invalidated references statically by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2037
* Remove Restricted Type from Cadence by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2596
* Remove `pub`, `priv` and `pub(set)` access modifiers by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2540
* View functions:
    * Add machinery for parsing and checking `view` functions by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1818
    * View Enforcement by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1870
    * Enforce `view` in function conditions by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1871
* Remove deprecated key management API by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1842
* Fix ToBigEndianBytes implementations for U?Int(128|256) by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/1917
* Improve reference expressions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2010
* Take argument labels into account when declaring members for nested declarations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2013
* Improve resource tracking for optional binding by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2044
* Improve the definite return analysis by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2090
* Change for-loop variables from per-loop to per-iteration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2024
* Reject cyclic imports during contract deployment by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2308
* Parser/syntax changes:
    * Prevent keywords from being used as identifiers (redux) by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/1937
    * Prevent keywords from being used as names in function declarations by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2067
    * Function type syntax change by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2140
    * Report preceding identifier in member declaration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2231

### ⭐ Features
* Add interface inheritance by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2112
* Attachment Iteration by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2440
* Convert generic container types on upcast by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2567
* Allow emit statements in conditions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2589

### 🛠 Improvements
* Add invalidation info to the invalid reference usage error by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2096
* Improve view modifier parsing and printing by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2117
* Make view a soft keyword by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2121
* Add more purity annotations to built-in and standard library function types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2125
* Export all keywords by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2131
* Add mapping function for types by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2425
* Improve formatting of access modifiers by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2597
* Improve cyclic import handling by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2632

### 🐞 Bug Fixes
* Fix test matcher by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2130
* Update Stable Cadence branch, fix view function expression parsing by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2141
* Transaction pre-conditions must also be `view` by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2171
* fix view parsing for expressions by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2164
* Add attachment keywords to keyword lists by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2499
* Fix crash on entitlement mappings with empty outputs by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2592
* Make more `AuthAccount` functions `view` by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2666
* Fix entitlement memory metering by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2668
* Fix interface inheritance for nested interfaces by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2674
* Update reference invalidation check to consider optional resources by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2680

### Other Changes
* Refactor reference tests by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2086
* Clean up function types and type annotations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2126
* Optimize keyword testing by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2295
* backport [#2302](https://github.com/onflow/cadence/issues/2302) to stable cadence (fix docstring parsing after functions) by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2329
* Add tests for entitlement inheritance between interfaces by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2564
* Add cyclic imports tests by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2629


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.40.0...v0.41.0-stable-cadence.1

[Changes][v0.41.0-stable-cadence.1]


<a name="v0.40.0"></a>
# [v0.40.0](https://github.com/onflow/cadence/releases/tag/v0.40.0) - 03 Aug 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.40.0 -->

## What's Changed

### ⭐ Features
* Add utility function `ccf.HasMsgPrefix` by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2609
* Introduce `Character.utf8` by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2617
* Provide suggested fixes for argument label errors by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2619
* Suggest optional chaining by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2625
* Introduce `reverse` in Fixed/Variable sized Array type by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2654

### 🛠 Improvements
* Improve update tool config by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2601
* Update integration tests for built-in `Crypto` contract by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2615
* Use atree orderd map's `Has` function to test for existence by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2618
* Delegate Cadence composite `Storable()` to Atree by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2621
* Delegate storage address to Atree by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2622
* Cleanup by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2692
* Temporarily remove deprecations for existing capability API by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2673

### 🐞 Bug Fixes
* Populate `Message`, `LocationRange` fields for `AssertionError` on `Test.expect` function by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2607
* Fix data race in sema/type_test.go by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2624
* Fix potential data races by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2633
* Add support for max parameter count (internal [#128](https://github.com/onflow/cadence/issues/128)) by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2656
* Remove spurious resource loss errors (internal [#130](https://github.com/onflow/cadence/issues/130)) by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2657
* Fix checking of reference expressions involving optionals (internal [#129](https://github.com/onflow/cadence/issues/129)) by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2658
* Prevent recursive transfers (internal [#133](https://github.com/onflow/cadence/issues/133)) by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2659
* Handle destroyed optional references by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2661
* Fix positioning of default function conflict errors by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2665
* Wrap host env errors with external errors by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2683
* Fix storage iteration and error misclassification by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2676
* Fix error on reference creation with invalid type by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2687

### Other Changes
* Add support for forks to compatibility check by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2612
* Set permissions in backward compatibility check by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2613
* Don't skip benchmark for forks by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2616
* Add mailmap by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2628
* Use diff command to generate diff by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2636



**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.14...v0.40.0

[Changes][v0.40.0]


<a name="v0.39.14"></a>
# [v0.39.14](https://github.com/onflow/cadence/releases/tag/v0.39.14) - 11 Jul 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.14 -->

## What's Changed
### 🐞 Bug Fixes
* Fix resource tracking by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2568
* [v0.39] Add support for max parameter count (internal [#128](https://github.com/onflow/cadence/issues/128)) by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2650
* [v0.39] Remove spurious resource loss errors (internal [#130](https://github.com/onflow/cadence/issues/130)) by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2651
* [v0.39] Fix checking of reference expressions involving optionals (internal [#129](https://github.com/onflow/cadence/issues/129)) by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2652
* [v0.39] Prevent recursive transfers (internal [#133](https://github.com/onflow/cadence/issues/133)) by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2653
### Other Changes
* Add tests for resource tracking in bound function invocations by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2569
* Merge `release/v0.39.12` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2591


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.12...v0.39.14

[Changes][v0.39.14]


<a name="v0.39.13-stable-cadence"></a>
# [v0.39.13-stable cadence (v0.39.13-stable-cadence)](https://github.com/onflow/cadence/releases/tag/v0.39.13-stable-cadence) - 26 Jun 2023

This is a preview release of [Stable Cadence](https://forum.onflow.org/t/the-path-to-stable-cadence/2702).

Read all about this release in the forum announcement (link forthcoming). 

## What's Changed
### 💥 Language Breaking Changes
* Entitlements
* Remove `pub`, `priv` and `pub(set)` access modifiers by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2540
### ⭐ Features
* Add interface inheritance by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2112
* Attachment Iteration by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2440

[Changes][v0.39.13-stable-cadence]


<a name="v0.39.12"></a>
# [v0.39.12](https://github.com/onflow/cadence/releases/tag/v0.39.12) - 21 Jun 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.12 -->

## What's Changed
### 🐞 Bug Fixes
* Port internal fix 126 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2590


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.11...v0.39.12

[Changes][v0.39.12]


<a name="v0.39.11"></a>
# [v0.39.11](https://github.com/onflow/cadence/releases/tag/v0.39.11) - 20 Jun 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.11 -->

## What's Changed
### 🐞 Bug Fixes
* Fix panic in `DecodeFields` on bad inputs by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2584

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.10...v0.39.11

[Changes][v0.39.11]


<a name="v0.39.10"></a>
# [v0.39.10](https://github.com/onflow/cadence/releases/tag/v0.39.10) - 17 Jun 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.10 -->

## What's Changed
### ⭐ Features
* Add utility to decode fields into struct based on field tags by [@rrrkren](https://github.com/rrrkren) in https://github.com/onflow/cadence/pull/2565
* Introduce the `Test.assertEqual` function by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2571
* Add the `CoverageReport.MarshalLCOV` method to generate LCOV format by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2577
* decode complex fields into more primitive types by [@rrrkren](https://github.com/rrrkren) in https://github.com/onflow/cadence/pull/2572
### 🐞 Bug Fixes
* Fix resource tracking by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2578

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.8...v0.39.10

[Changes][v0.39.10]


<a name="v0.39.9"></a>
# [v0.39.9](https://github.com/onflow/cadence/releases/tag/v0.39.9) - 16 Jun 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.9 -->

## What's Changed
### 🐞 Bug Fixes
* Fix resource tracking by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2568

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.8...v0.39.9

[Changes][v0.39.9]


<a name="v0.39.8"></a>
# [v0.39.8](https://github.com/onflow/cadence/releases/tag/v0.39.8) - 12 Jun 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.8 -->

## What's Changed
### 🐞 Bug Fixes
* Prevent function invocation on destroyed resource by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2561

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.7...v0.39.8

[Changes][v0.39.8]


<a name="v0.39.7"></a>
# [v0.39.7](https://github.com/onflow/cadence/releases/tag/v0.39.7) - 12 Jun 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.7 -->

## What's Changed
### ⭐ Features
* Add support for CCF encoding/decoding modes and options by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2559
### 🐞 Bug Fixes
* Fix race conditions in test by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2556


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.6...v0.39.7

[Changes][v0.39.7]


<a name="v0.39.6"></a>
# [v0.39.6](https://github.com/onflow/cadence/releases/tag/v0.39.6) - 09 Jun 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.6 -->

## What's Changed
### ⭐ Features
* Add support for suggested fixes to analysis diagnostics by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2550
* Fetch event fields by name by [@rrrkren](https://github.com/rrrkren) in https://github.com/onflow/cadence/pull/2554
### 🛠 Improvements
* Adding contract init as entry points by [@bthaile](https://github.com/bthaile) in https://github.com/onflow/cadence/pull/2553
### 🐞 Bug Fixes
* Improve function type decoding by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2555
### Other Changes
* Merge `release/v0.39.5` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2552


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.5...v0.39.6

[Changes][v0.39.6]


<a name="v0.39.5"></a>
# [v0.39.5](https://github.com/onflow/cadence/releases/tag/v0.39.5) - 08 Jun 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.5 -->

## What's Changed

### 🐞 Bug Fixes
* Bring back type IDs in JSON encoding of function types and restricted types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2551


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.4...v0.39.5

[Changes][v0.39.5]


<a name="v0.39.4"></a>
# [v0.39.4](https://github.com/onflow/cadence/releases/tag/v0.39.4) - 07 Jun 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.4 -->

## What's Changed
### 🐞 Bug Fixes

* Fix test framework: Pass interpreter when iterating over arrays by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2547

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.3...v0.39.4

[Changes][v0.39.4]


<a name="v0.39.3"></a>
# [v0.39.3](https://github.com/onflow/cadence/releases/tag/v0.39.3) - 07 Jun 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.3 -->

## What's Changed
### 🐞 Bug Fixes
* Add function type to function wrapper host functions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2542
* Port internal fix 119 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2544
* Port internal fix 120 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2545

### Other Changes
* Update configuration of update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2539
* Update config of update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2543


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.2...v0.39.3

[Changes][v0.39.3]


<a name="v0.39.2"></a>
# [v0.39.2](https://github.com/onflow/cadence/releases/tag/v0.39.2) - 06 Jun 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.2 -->

## What's Changed
### ⭐ Features
* Add composite type handler, allow loading of `flow.*` event types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2533
### 🛠 Improvements
* Return a text message instead of NaN by [@kozlovb](https://github.com/kozlovb) in https://github.com/onflow/cadence/pull/2511
### 🐞 Bug Fixes
* Fix  `publicKey` to have matching static and runtime types by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2528
* Fix `codeHash` types to match (static and runtime) by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2529
### Other Changes
* Merge `release/v0.39.1` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2522
* Check code hash value's type is correct by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2532

## New Contributors
* [@kozlovb](https://github.com/kozlovb) made their first contribution in https://github.com/onflow/cadence/pull/2511

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.1...v0.39.2

[Changes][v0.39.2]


<a name="v0.31.6"></a>
# [v0.31.6](https://github.com/onflow/cadence/releases/tag/v0.31.6) - 06 Jun 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.31.6 -->

## What's Changed
### 🛠 Improvements
* Enforce account links to be private by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2369
* Emit an event when an account is linked by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2370
* Change account linking pragma to run-time configuration of execution by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2383
* Require #allowAccountLinking pragma to allow use of account linking function by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2359
### 🐞 Bug Fixes
* Check cyclic storage references during export by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2408
* Check for potential resource loss in reference expression by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2436

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.31.5...v0.31.6

[Changes][v0.31.6]


<a name="v0.39.1"></a>
# [v0.39.1](https://github.com/onflow/cadence/releases/tag/v0.39.1) - 31 May 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.1 -->

## What's Changed

### ⭐ Features
* Support cyclic reference value in CCF codec by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2521

### Other Changes

* Add rule to clean linter by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2508


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.39.0...v0.39.1

[Changes][v0.39.1]


<a name="v0.39.0"></a>
# [v0.39.0](https://github.com/onflow/cadence/releases/tag/v0.39.0) - 31 May 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.39.0 -->

## What's Changed

### ⭐ Features

### Capability Controllers

[FLIP 798: Capability controllers](https://github.com/onflow/flips/blob/main/cadence/20220203-capability-controllers.md) got implemented.

You can start migrating existing uses of the linking-based capability API to the new Capability Controller API. The linking-based capability API is now deprecated and will be removed in a future release.

* Capability controllers by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2501

### Compact Cadence Format

A codec for the new [Cadence Compact Format (CCF)](https://github.com/fxamacker/ccf_draft) got implemented,
a more efficient format for encoding Cadence values. It is an alternative to the current [JSON-Cadence](https://developers.flow.com/cadence/json-cadence-spec) format, and will eventually replace it.

* Add CCF codec implementing CCF Specification (RC1) by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2364
* Support 2 new attachment types in CCF codec by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2516

### Standard Library

Thanks to [a Developer Grant](https://github.com/onflow/developer-grants/issues/179), new contributor [@darkdrag00nv2](https://github.com/darkdrag00nv2) improved the standard library with many new features: 👏 

* Add `Address.fromBytes` to create `Address` from `[UInt8]` by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2452
* Extend ordering comparisons (<, <=, >, >=) to `Bool` and `Character` by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2430
* Extend ordering comparison (<, <=, >, >=) to `String` by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2470
* Allow equality comparisons for dictionaries by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2414
* Implement `Address.fromString` by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2491
* Add `fromBigEndianBytes` conversion function to `Number` types by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2496
* Introduce `Word128` integer type by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2497
* Introduce `Word256` integer type by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2510
* Implement `check` functionality in AuthAccount by [@darkdrag00nv2](https://github.com/darkdrag00nv2) in https://github.com/onflow/cadence/pull/2492

### Tooling

* Introduce the `Test.expectFailure` function by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2468
* Enrich test framework interface by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2505
* Introduce a `LocationFilter` field on `CoverageReport` type by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2474
* Add basic support for multiline input to REPL by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2434
* Add history support to REPL by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2442

### 🛠 Improvements

* Improve external types and JSON codec by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2401
* Add support for generating code of nested types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2417
* Refactor `AuthAccount` and `PublicAccount` types to Cadence by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2418
* Refactor and optimize types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2419
* Update handling of user errors returned by atree by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2426
* Add more built-in matchers to the Cadence Testing Framework by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2420
* Add reference tracking for the `result` variable by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2447
* Various improvements by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2451
* Refactor console REPL into type, add type command and command completion by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2450
* Initialize `stdlib` values of `Test` contract lazily by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2403
* Handle wrapped errors during storage iteration by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2488
* Improve code generation for members by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2489
* Improve errors by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2502
* Add tests for test framework blockchain by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2514
* Improve export and encoding by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2517
* Add support for restricted types to Go code generator by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2475

### 🐞 Bug Fixes

* Check for potential resource loss in reference expression by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2437
* Fix built-in `result` variable for optional resource typed returns by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2444
* Add validation for exporting references to destroyed resources by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2453
* Fix and test sema to static type conversion by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2465
* Fix race conditions in CCF tests caused by shared usage of types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2467
* Properly wrap nil in optionals depending on the nested optional type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2469
* Don't truncate error message stack early in case of multiline errors by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2486
* Fix userPanicToError to handle wrapped user error. by [@pattyshack](https://github.com/pattyshack) in https://github.com/onflow/cadence/pull/2498

### 📖 Documentation

* Remove parameter name from API documentation when argument label is given by [@MaxStalker](https://github.com/MaxStalker) in https://github.com/onflow/cadence/pull/2406
* Update crypto doc by [@tarakby](https://github.com/tarakby) in https://github.com/onflow/cadence/pull/2439
* Improve documentation sidebar by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2456
* Add an `Examples` section on the Cadence Testing Framework doc by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2493
* replace docs with links by [@nialexsan](https://github.com/nialexsan) in https://github.com/onflow/cadence/pull/2487

### 💥 Go API Breaking Chance

* Improve `cadence.Path` by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2427
* Introduce a `CoverageReport` field on `runtime.Config` type by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2416

### Other Changes

* Update dependencies by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2407
* Fix release notes configuration  by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2435
* Add integration tests (in Cadence) for `Crypto` contract and generate code coverage by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2485
* Improve linting by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2512

## New Contributors

* [@darkdrag00nv2](https://github.com/darkdrag00nv2) made their first contribution in https://github.com/onflow/cadence/pull/2414
* [@nialexsan](https://github.com/nialexsan) made their first contribution in https://github.com/onflow/cadence/pull/2487

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.38.0...v0.39.0

[Changes][v0.39.0]


<a name="v0.38.1"></a>
# [v0.38.1](https://github.com/onflow/cadence/releases/tag/v0.38.1) - 12 Apr 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.38.1 -->

## What's Changed

### 🐞 Bug Fixes

* Check for potential resource loss in reference expression by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2438

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.38.0...v0.38.1

[Changes][v0.38.1]


<a name="v0.38.0"></a>
# [v0.38.0](https://github.com/onflow/cadence/releases/tag/v0.38.0) - 31 Mar 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.38.0 -->

## What's Changed
### ⭐ Features
* Improvements to `CoverageReport` public API by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2391
### 🛠 Improvements
* Allow empty qualified identifier when decoding type IDs by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2396
### 🐞 Bug Fixes
* Generate labels and identifiers for function parameters in Go code generator by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2400
* Check cyclic storage references during export by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2409
### 📖 Documentation
* Revised and extended documentation of `unsafeRandom` by [@AlexHentschel](https://github.com/AlexHentschel) in https://github.com/onflow/cadence/pull/2399

## New Contributors
* [@AlexHentschel](https://github.com/AlexHentschel) made their first contribution in https://github.com/onflow/cadence/pull/2399

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.37.0...v0.38.0

[Changes][v0.38.0]


<a name="v0.37.0"></a>
# [v0.37.0](https://github.com/onflow/cadence/releases/tag/v0.37.0) - 23 Mar 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.37.0 -->

## What's Changed

### ⭐ Features

* The new AuthAccount capabilities ([FLIP 53](https://github.com/onflow/flips/pull/53)) feature is now feature complete 
* Add a Reset method on CoverageReport type by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2386

### 🛠 Improvements
* Change account linking pragma to run-time configuration of execution by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2380
* test: simplify TestCrashers by [@Juneezee](https://github.com/Juneezee) in https://github.com/onflow/cadence/pull/2378
* Fix Value.Type() returning Type referencing nil ptr for eight external values by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2388
* Use `AddressLocation` in `Interface` by [@janezpodhostnik](https://github.com/janezpodhostnik) in https://github.com/onflow/cadence/pull/2382

### 🐞 Bug Fixes
* Revert "Remove support for importing/exporting and JSON encoding/deco… by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2381
* Improve contract address for code coverage report by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2372
* Fix and run interpreter smoke tests on CI by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2395

### 📖 Documentation
* Document how to create a GitHub release by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2375
* Fix release note for attachments by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2393

### Other Changes
* Merge `release/v0.36.0` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2374
* Update golangci-lint to latest release, which allows use of Go 1.20 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2376
* Fix broken account tests by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2384

## New Contributors
* [@Juneezee](https://github.com/Juneezee) made their first contribution in https://github.com/onflow/cadence/pull/2378

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.36.0...v0.37.0

[Changes][v0.37.0]


<a name="v0.36.0"></a>
# [v0.36.0](https://github.com/onflow/cadence/releases/tag/v0.36.0) - 08 Mar 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.36.0 -->

## What's Changed
### ⭐ Features
* Attachments (with a feature flag to enable) by [@dsainati1](https://github.com/dsainati1) in multiple PRs
* Require #allowAccountLinking pragma to allow use of account linking function by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2355
* Improve errors by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2356
* Enforce account links to be private by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2366
* Emit an event when an account is linked by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2368
* Add metering for encoded values by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2339

### 🐞 Bug Fixes
* Fix racy tests by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2345
* Code coverage support for pre/post conditions inside functions by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2371
* Add position information to rewritten post condition before statement by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2373

### 📖 Documentation
* Attachments Documentation by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2137
* Add meeting notes for Jan + Feb 2023 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2347
* fixed formating of codeblocks in lists and remove outdated callout  by [@bjartek](https://github.com/bjartek) in https://github.com/onflow/cadence/pull/2349

### Other Changes
* * Rename GetAndSetProgram to GetOrLoadProgram by [@janezpodhostnik](https://github.com/janezpodhostnik) in https://github.com/onflow/cadence/pull/2362


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.35.0...v0.36.0

[Changes][v0.36.0]


<a name="v0.35.0"></a>
# [v0.35.0](https://github.com/onflow/cadence/releases/tag/v0.35.0) - 24 Feb 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.35.0 -->

## What's Changed

### ⭐ Features
* If extended elaboration is enabled, record all expression types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2340

### 🛠 Improvements
* Use pointer receivers for external types with cached typeIDs by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2336
* Refactor tests to cache typeIDs during assertions by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2328
* Cache external Cadence type IDs by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2326

### 🐞 Bug Fixes
* Convert reference values by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2342

### 📖 Documentation
* Update docstrings of borrow functions to reflect current behaviour by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2335


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.34.0...v0.35.0

[Changes][v0.35.0]


<a name="v0.31.5"></a>
# [v0.31.5](https://github.com/onflow/cadence/releases/tag/v0.31.5) - 23 Feb 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.31.5 -->

## What's Changed

### 🐞 Bug Fixes

* Convert reference values by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2341


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.31.3...v0.31.5

[Changes][v0.31.5]


<a name="v0.34.0"></a>
# [v0.34.0](https://github.com/onflow/cadence/releases/tag/v0.34.0) - 16 Feb 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.34.0 -->

## What's Changed
### 🛠 Improvements
* Improve code coverage functionality by [@m-Peter](https://github.com/m-Peter) in https://github.com/onflow/cadence/pull/2296
* swap arguments of NewHostFunctionValue to put the fn type first by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2323
* Handle backticks in docstrings when generating code, add surrounding newlines to docstrings by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2330
### 🐞 Bug Fixes
* lift getLinkTargetFunction into parent scope to cache it by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2332
### 📖 Documentation
* Minor clarification for release docs by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/2289
### Other Changes
* Merge `release/v0.33.0` to `master` by [@github-actions](https://github.com/github-actions) in https://github.com/onflow/cadence/pull/2317
* Run compatibility checker for PRs by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2318

## New Contributors
* [@m-Peter](https://github.com/m-Peter) made their first contribution in https://github.com/onflow/cadence/pull/2296

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.33.0...v0.34.0

[Changes][v0.34.0]


<a name="v0.33.0"></a>
# [v0.33.0](https://github.com/onflow/cadence/releases/tag/v0.33.0) - 08 Feb 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.33.0 -->

## What's Changed
### ⭐ Features
* Add publicTypes method to DeployedContract for introspection by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2242
* Parse type parameter lists for function declarations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2145
* Allow equality tests on arrays if their inner types support it by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2271
### 🛠 Improvements
* improve wording and classification of some interpreter errors by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2194
* Also update flow-go's new insecure package by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2220
* Correct example code link by [@alilloig](https://github.com/alilloig) in https://github.com/onflow/cadence/pull/2209
* Allow errors to pretty-print excerpts over multiple lines by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2197
* Add additional metering methods to the interface by [@janezpodhostnik](https://github.com/janezpodhostnik) in https://github.com/onflow/cadence/pull/2222
* Ensure GetProgram calls before SetProgram by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2249
* Replace interface GetProgram and SetProgram with GetAndSetProgram by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2254
* Replace empty common.Address{} with singleton by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2256
* Improve update tool and documentation by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2261
* add unwrap to ExternalError by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2286
* Refactor external types to instantiate only once whenever possible by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2225
* add Config() method to runtime.Runtime interface by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2281
* Improve compatibility diff checker by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2291
* Add Cadence to Go type translator, refactor Block type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2278
* Add support for type parameters to Cadence to Go type translator by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2287
* Simplify lookaheads by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2298
* Clear the test runtime program cache automatically after contract update by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2306
* Use dedicated type parameters for test-stdlib functions by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2206
* Remove support for importing/exporting and JSON encoding/decoding path links and cadence type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2248
* Add backward compatibility check CI workflow by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2237
* Avoid unnecessary use of any and casts, parse function is generic now by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2270
* Automate the release process by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2239
* Make compatibility checker workflow work with old tags by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2288
* Improve version bump command by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2314
### 🐞 Bug Fixes
* Revert "don't ignore invalid identifier" / PR  by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2218
* Ensure SetProgram is called after GetProgram, even if there is an error by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2245
* Fix loop variables being captured in closures by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2255
* Re-merge [#2302](https://github.com/onflow/cadence/issues/2302), Fix docstring parsing for functions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2313
* Retain docstrings for special function declarations, remove unnecessary return type annotations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2303
* Handle cyclic imports in analyzer tool by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2315
### 📖 Documentation
* Add documentation for contracts.borrow by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2216
* Remove old diagrams by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2246
* Fix Capability example in JSON-Cadence Data Interchange Format specs by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/2207
* Add documentation on the release process by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2253
* Update docs to reflect Account inbox live after MN21 spork by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/2265
* Updates to Cadence guidance for Solidity developers by [@franklywatson](https://github.com/franklywatson) in https://github.com/onflow/cadence/pull/2263
* Fix: typos by [@omahs](https://github.com/omahs) in https://github.com/onflow/cadence/pull/2297
### Other Changes
* Bump version by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2214
* Refactor the existing issue templates to forms by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2229
* Remove copyright years from file headers, update years in NOTICE by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2258
* Downgrade benchstat version to bring back html output by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2274
* Clear warnings by removing unnecessary type arguments in tests by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2277
* Fix confusion of return type of checker's visitExpression methods by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2269
* add dreamsmasher to issue assignees, codeowners for automatic reviewer requests by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2312
* Bypass go-proxy to always pick the latest commit by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2316

## New Contributors
* [@franklywatson](https://github.com/franklywatson) made their first contribution in https://github.com/onflow/cadence/pull/2263
* [@omahs](https://github.com/omahs) made their first contribution in https://github.com/onflow/cadence/pull/2297

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.31.1...v0.33.0

[Changes][v0.33.0]


<a name="v0.31.3"></a>
# [v0.31.3](https://github.com/onflow/cadence/releases/tag/v0.31.3) - 13 Jan 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.31.3 -->

## What's Changed

### 🐞 Bug Fixes
* Ensure SetProgram is called after GetProgram, even if there is an error by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2241


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.31.2...v0.31.3

[Changes][v0.31.3]


<a name="v0.31.2"></a>
# [v0.31.2](https://github.com/onflow/cadence/releases/tag/v0.31.2) - 05 Jan 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.31.2 -->

## What's Changed

### 🐞 Bug Fixes
* Revert "don't ignore invalid identifier" / PR by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2218

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.31.1...v0.31.2

[Changes][v0.31.2]


<a name="v0.31.1"></a>
# [v0.31.1](https://github.com/onflow/cadence/releases/tag/v0.31.1) - 04 Jan 2023

<!-- Release notes generated using configuration in .github/release.yml at v0.31.1 -->

## What's Changed
### 🛠 Improvements
* Add equal method for cadence external types by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2203
### 🐞 Bug Fixes
* Fix function activations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2213
### 📖 Documentation
* Correct definition of block view by [@jordanschalm](https://github.com/jordanschalm) in https://github.com/onflow/cadence/pull/2205

## New Contributors
* [@jordanschalm](https://github.com/jordanschalm) made their first contribution in https://github.com/onflow/cadence/pull/2205

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.31.0...v0.31.1

[Changes][v0.31.1]


<a name="v0.31.0"></a>
# [v0.31.0](https://github.com/onflow/cadence/releases/tag/v0.31.0) - 22 Dec 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.31.0 -->

## What's Changed
### 🛠 Improvements
* Adds some suggestions/improvements to sema error messages by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2190
* Skip broken values during storage iteration by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2202
### Other Changes
* Improve field alignment by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2186
* Add update tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2198
* Add test case for publishing and claiming private account capability by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2195


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.30.0...v0.31.0

[Changes][v0.31.0]


<a name="v0.30.0"></a>
# [v0.30.0](https://github.com/onflow/cadence/releases/tag/v0.30.0) - 13 Dec 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.30.0 -->

## What's Changed

### ⭐ Features
* Improve JSON encoding test cases, add REPL command for exporting by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2107
* Add static and native modifiers for fields and functions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2119
* Implement [FLIP 1071](https://github.com/onflow/flow/pull/1071): borrow contract by [@bluesign](https://github.com/bluesign) in https://github.com/onflow/cadence/pull/1934
* Allow embedders to reuse interpreter shared state by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2100

### 🛠 Improvements
* Improve interpreter config by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2097
* Allow export of function values by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2052
* Improve JSON decoding error messages, add utility by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2104
* Improve common supertype calculation by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2103
* Fix "log" crashing the REPL by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2109
* Type fixes and improvements by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2135
* Improve error messages for values/types that cannot be imported or exported by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2138
* Meter big int allocations in parser, improve over-estimation by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2110
* Improve parameter AST node by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2143
* Address some tech debt by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2155
* Improve reference static type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2156
* Improve type mismatch errors by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2161
* Improve sema by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2166
* Optimize sema by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2162
* Use invocation's interpreter by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2159
* improve ConformanceError error message by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2172
* Suggest lexically closest member in cases of NotDeclaredMemberError by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2173
* Add support for nested pragma declarations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2169
* Initialize the Crypto contract lazily by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2183
* Pretty print interpreter errors by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2185
* Fix whitespace parsing between expressions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2115
* Improve allocations in checker by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2127
* Add positions to numerical runtime errors by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2174
* Optimize elaboration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2178
* Avoid unnecessary slice allocations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2191

### 🐞 Bug Fixes
* Fix self declaration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2108
* Report invalid identifier preceding a member or nested declaration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2134
* Fix parsing of array and dictionary expressions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2136
### 📖 Documentation
* rename why-cadence.md to why.md to fix broken link by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/2102
* Add documentation for String.fromUTF8, numeric types' fromString by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2105
* Note in docs that inbox api is not yet released on mainnet by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2123
* Fix documentation by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2122
* Remove outdated callout from the testing-framework docs by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2132
* Update flow-docs.json by [@bthaile](https://github.com/bthaile) in https://github.com/onflow/cadence/pull/2133
* Add meeting notes for interface inheritance meeting by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2144
* Update design-patterns.mdx by [@gregsantos](https://github.com/gregsantos) in https://github.com/onflow/cadence/pull/2146
* Add link to style guide by [@bthaile](https://github.com/bthaile) in https://github.com/onflow/cadence/pull/2149
* Fix links in documentation by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2142

### Other Changes

* Add JSON language annotations to expected JSON encodings in AST tests by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2118
* Move test framework tests to Cadence by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2129
* Update CI jobs by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2128
* Lint JSON files by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2139
* Clean up metering by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2152
* Report unkeyed composite literals by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2184

## New Contributors
* [@bthaile](https://github.com/bthaile) made their first contribution in https://github.com/onflow/cadence/pull/2133
* [@gregsantos](https://github.com/gregsantos) made their first contribution in https://github.com/onflow/cadence/pull/2146

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.29.0...v0.30.0

[Changes][v0.30.0]


<a name="v0.29.0"></a>
# [v0.29.0](https://github.com/onflow/cadence/releases/tag/v0.29.0) - 27 Oct 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.29.0 -->

## What's Changed

### ⭐ Features
* Publish and Claim: Inbox API for Capability Bootstrapping by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1983
* Allow for-loops over strings by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2080
* Add `String.fromCharacters` by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2083

### 🛠 Improvements
* Directly transfer published values by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2042
* Simplify parser by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2064
* Re-use interpreter for value decoding in test-framework by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2074
* Use Value.Clone instead of custom deep copy function by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2058
* Improve interpreter storage by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2072
* Improve type inference for case expressions in switch statement by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2079
* refactor AccountKeysCount to expect an error too by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2093
* Make the interpreter globals map allocation lazy by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2065
* Improve block comment parsing by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2068
* Optimize resource tracking by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2073

### 🐞 Bug Fixes
* Fix account inbox implementation by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2053

### 📖 Documentation
* Update design pattern documentation to refer to the new Capability Bootstrapping API by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2035
* Fixes dead link in unsaferandom documentation by [@justjoolz](https://github.com/justjoolz) in https://github.com/onflow/cadence/pull/2047
* cleanup docs-tutorial06 by [@AmarildoGrembi](https://github.com/AmarildoGrembi) in https://github.com/onflow/cadence/pull/2071
* Document setup() and tearDown() functions in test-framework by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2043
* Add "why Cadence" section to Cadence docs by [@j1010001](https://github.com/j1010001) in https://github.com/onflow/cadence/pull/2039
* Update flow-docs.json to include "why cadence" section. by [@10thfloor](https://github.com/10thfloor) in https://github.com/onflow/cadence/pull/2040
* Adds link to Playground on Cadence homepage by [@10thfloor](https://github.com/10thfloor) in https://github.com/onflow/cadence/pull/2078

### 💥 Breaking Changes
* Add key iteration, count methods to AccountKeyProvider interface by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2038
  (This change does not affect Cadence programs, only embedders, like FVM)

### Other Changes
* Lint with gofmt 1.19, upgrade to go 1.19 by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2041
* Improve CI by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2055
* Parallelize more tests, use checker errors helper by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2087


## New Contributors
* [@justjoolz](https://github.com/justjoolz) made their first contribution in https://github.com/onflow/cadence/pull/2047
* [@AmarildoGrembi](https://github.com/AmarildoGrembi) made their first contribution in https://github.com/onflow/cadence/pull/2071
* [@j1010001](https://github.com/j1010001) made their first contribution in https://github.com/onflow/cadence/pull/2039

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.28.0...v0.29.0

[Changes][v0.29.0]


<a name="v0.29.0-stable-cadence"></a>
# [v0.29.0-stable-cadence](https://github.com/onflow/cadence/releases/tag/v0.29.0-stable-cadence) - 27 Oct 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.29.0-stable-cadence -->

This is a preview release of [Stable Cadence](https://forum.onflow.org/t/the-path-to-stable-cadence/2702).

Read all about this release in the [forum announcement](https://forum.onflow.org/t/another-update-on-stable-cadence/3715).

## What's Changed

### 💥 Breaking Changes
* Remove deprecated key management API by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1842
* Fix ToBigEndianBytes implementations for U?Int(128|256) by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/1917
* Prevent keywords from being used as identifiers (redux) by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/1937
* Add machinery for parsing and checking `view` functions by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1818
* View Enforcement by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1870
* Enforce `view` in function conditions by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1871
* Improve reference expressions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2010
* Take argument labels into account when declaring members for nested declarations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2013
* Improve resource tracking for optional binding by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2044
* Detect invalidated references statically by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2037
* Invalidate resource references when the referenced resource is moved by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1999

### 🛠 Improvements
* Prevent keywords from being used as names in function declarations by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2067
* Add invalidation info to the invalid reference usage error by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2096

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.29.0...v0.29.0-stable-cadence

[Changes][v0.29.0-stable-cadence]


<a name="v0.28.0"></a>
# [v0.28.0](https://github.com/onflow/cadence/releases/tag/v0.28.0) - 06 Oct 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.28.0 -->

## What's Changed

### ⭐ Features
* Add fromString method on numeric types by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/1982

### 🛠 Improvements
* Improve resource tracking by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2033
* Catch interpreter panics in REPL by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2028
* Improve REPL by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2021
* Introduce and use singleton for Void value by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2029
* Introduce and use singletons for bool values by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2030
* Introduce and use singleton value for nil by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2031
* Optimize location range by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2036
* Check errors' potential secondary message and error notes by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2034

### 📖 Documentation
* Update 06-fungible-tokens.mdx by [@MrDSGC](https://github.com/MrDSGC) in https://github.com/onflow/cadence/pull/2020
* Fix composite types documentation by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2025

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.27.1...v0.28.0

[Changes][v0.28.0]


<a name="v0.27.1"></a>
# [v0.27.1](https://github.com/onflow/cadence/releases/tag/v0.27.1) - 03 Oct 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.27.1 -->

## What's Changed
### ⭐ Features
* Add key iteration method on dictionaries by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/1992
* Make Void values constructible by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/2004

### 🛠 Improvements
* Improve parser function naming, remove dead code by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/2016
### 🐞 Bug Fixes
* Fix accounts not being returnable from scripts by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2022
### 📖 Documentation
* Add language clarifying the type parameter's type in the private and public path iteration functions by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/2012
### Other Changes
* Remove npm-packages for the languageserver and docgen tool by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2002
* Make argument names in test-stdlib compatible with stable cadence by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2015


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.27.0...v0.27.1

[Changes][v0.27.1]


<a name="v0.27.0"></a>
# [v0.27.0](https://github.com/onflow/cadence/releases/tag/v0.27.0) - 23 Sep 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.27.0 -->

## What's Changed

This release contains a brand new framework for writing tests in Cadence, new standard library features, and many performance improvements.

### ⭐ Features
* Add testing framework for Cadence by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1923
* Make paths equatable by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/1911
* Add String.fromUTF8 for conversion from [UInt8] by [@dreamsmasher](https://github.com/dreamsmasher) in https://github.com/onflow/cadence/pull/1927

### 🛠 Improvements
* Optimize and make visitor generic by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1907
* Improve the AST visitor further by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1919
* Remove values from tokens, make all code byte slices by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1930
* Use WriteByte instead of WriteRune by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1944
* Export environment checker and interpreter config to allow customization by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1910
* Refactor activations to be truly generic by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1945
* Move the `fail()` function to native implementation by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1960
* Update wasmtime-based VM by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1946
* Optimize type ID generation, remove location ID by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1932
* Export functionality required by debugger by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1924
* Add method to get function value's sema function type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1991
* Improve type information in interpreter errors by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1994
* [LS] General improvements  by [@sideninja](https://github.com/sideninja) in https://github.com/onflow/cadence/pull/1877
* [LS] Update to Cadence v0.26.0 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1970
* [Test Framework] Remove test-framework field from interpreter by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1886
* [Test Framework] Move matcher implementation to cadence by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1895
* [Test Framework] Refactor and cleanup the test-stdlib by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1897
* [Test Framework] Start development of Cadence test framework by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1827
* [Test Framework] Add support for importing and initializing contracts by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1832
* [Test Framework] Use FVM as the environment provider for test runner by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1835
* [Test Framework] Add transaction execution support by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1849
* [Test Framework] Run setup/teardown before/after tests by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1864
* [Test Framework] Add support for loading programs from local files by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1869
* [Test Framework] Add helper function for contract deployment by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1868
* [Test Framework] Add initial support for Matchers by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1863
* [Test Framework] Provide a way to specify addresses for the imports by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1893
* [Test Framework] Update Cadence by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1955
* [Test Framework] Separate import resolver and file resolver by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1954

### 🐞 Bug Fixes
* Make missing casting expression a user error by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1914
* [LS] Multiple language server bugfixes by [@sideninja](https://github.com/sideninja) in https://github.com/onflow/cadence/pull/1950
* Fix command-line runner by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1915
* Fix unnecessarily specific cast in storage iteration by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1984
* Don't print large integers in error messages by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1987
* Track potential jumps, ignore resource invalidations which are only potential by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1979
* Fix batch script by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1996
* Define standard library values in analysis by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1997

### 📖 Documentation
* docs: add security best practices by [@alxflw](https://github.com/alxflw) in https://github.com/onflow/cadence/pull/1887
* Add description to interpreter.Value by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1920
* Docs: implement flow-docs.json to control sidebar and other docs content by [@schpet](https://github.com/schpet) in https://github.com/onflow/cadence/pull/1909
* Add meeting notes for August 30, 2022 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1928
* Clarify that updating a contract does not change stored data by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1953
* Fixed result given by slice by [@tromex](https://github.com/tromex) in https://github.com/onflow/cadence/pull/1957
* Update FAQ by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1947
* DOCS: Fix line about storable fields in composite types by [@alilloig](https://github.com/alilloig) in https://github.com/onflow/cadence/pull/1965
* update playground tutorials by [@MrDSGC](https://github.com/MrDSGC) in https://github.com/onflow/cadence/pull/1964
* Add meeting notes for extensions meeting by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1989
* fix error in Script-Accessible report example by [@gus-labs](https://github.com/gus-labs) in https://github.com/onflow/cadence/pull/1975
* Interface Default Functions status by [@ph0ph0](https://github.com/ph0ph0) in https://github.com/onflow/cadence/pull/1976
* [Test Framework] Add documentation for testing framework by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1900
* Add status callout for the test framework docs by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/2000

### 🧪 Testing

* Add test case to check optional-chaining argument evaluation order by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1925

### Chores
* Remove broken CI checks by [@schpet](https://github.com/schpet) in https://github.com/onflow/cadence/pull/1908
* Sync test-framework branch with master by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1921
* Remove test-framework module by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1963
* Remove outdated prettier plugin by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1967
* Remove languageserver module from the cadence repo by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1981
* Remove docgen module by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1980
* Improve linting errors on CI by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1998
* [LS] Release npm cadence-language-server v0.26.1 by [@sideninja](https://github.com/sideninja) in https://github.com/onflow/cadence/pull/1962
* [Test Framework] Sync test framework branch with master by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1829

## New Contributors
* [@schpet](https://github.com/schpet) made their first contribution in https://github.com/onflow/cadence/pull/1908
* [@tromex](https://github.com/tromex) made their first contribution in https://github.com/onflow/cadence/pull/1957
* [@MrDSGC](https://github.com/MrDSGC) made their first contribution in https://github.com/onflow/cadence/pull/1964
* [@gus-labs](https://github.com/gus-labs) made their first contribution in https://github.com/onflow/cadence/pull/1975
* [@ph0ph0](https://github.com/ph0ph0) made their first contribution in https://github.com/onflow/cadence/pull/1976

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.26.2...v0.27.0

[Changes][v0.27.0]


<a name="v0.26.2"></a>
# [v0.26.2](https://github.com/onflow/cadence/releases/tag/v0.26.2) - 23 Sep 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.26.2 -->

## What's Changed
### 🐞 Bug Fixes
* Fix unnecessarily specific cast in storage iteration by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1988

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.26.1...v0.26.2

[Changes][v0.26.2]


<a name="v0.25.2"></a>
# [v0.25.2](https://github.com/onflow/cadence/releases/tag/v0.25.2) - 20 Sep 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.25.2 -->



**Full Changelog**: https://github.com/onflow/cadence/compare/v0.25.1...v0.25.2

[Changes][v0.25.2]


<a name="v0.25.1"></a>
# [v0.25.1](https://github.com/onflow/cadence/releases/tag/v0.25.1) - 20 Sep 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.25.1 -->

## What's Changed
### Other Changes
* [v0.25] Track potential jumps, ignore resource invalidations which are only potential by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1978
* [v0.25] Don't print large integers in error messages by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1985


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.25.0...v0.25.1

[Changes][v0.25.1]


<a name="v0.26.1"></a>
# [v0.26.1](https://github.com/onflow/cadence/releases/tag/v0.26.1) - 16 Sep 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.26.1 -->

## What's Changed
### 🐞 Bug Fixes
* [v0.26] Fix command-line runner by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1977


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.26.0...v0.26.1

[Changes][v0.26.1]


<a name="languageserver/v0.26.2"></a>
# [Language Server v0.26.2 (languageserver/v0.26.2)](https://github.com/onflow/cadence/releases/tag/languageserver/v0.26.2) - 16 Sep 2022

## 🛠 Improvements
- Update to Cadence v0.26.0 [#1970](https://github.com/onflow/cadence/issues/1970) [@turbolent](https://github.com/turbolent) 

[Changes][languageserver/v0.26.2]


<a name="languageserver/v0.26.1"></a>
# [Language Server v0.26.1 (languageserver/v0.26.1)](https://github.com/onflow/cadence/releases/tag/languageserver/v0.26.1) - 13 Sep 2022

## 🐞 Bug Fixes
**Fixed known bugs**
Multiple bugfixes addressing checker cache reset, making changes in the imported contracts visible, fixing some edge case bugs and improving the handling of the parser errors [#1950](https://github.com/onflow/cadence/issues/1950) [@sideninja](https://github.com/sideninja) 


[Changes][languageserver/v0.26.1]


<a name="v0.26.0"></a>
# [v0.26.0](https://github.com/onflow/cadence/releases/tag/v0.26.0) - 23 Aug 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.26.0 -->

## What's Changed

### ⭐ Features
* Interface default implementation support by [@bluesign](https://github.com/bluesign) in https://github.com/onflow/cadence/pull/1076
* Add path fields to `AuthAccount` and `PublicAccount`  by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1845
* Storage Iteration functions for accounts by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1851
* Add panic after continuing storage iteration after a storage mutation by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1892

### 🛠 Improvements
* [LS] Enable argument init for add contract by [@sukantoraymond](https://github.com/sukantoraymond) in https://github.com/onflow/cadence/pull/1855
* Optimize standard library enums by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1813
* Optimize elaboration by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1844
* Ignore Go workspace files by [@sideninja](https://github.com/sideninja) in https://github.com/onflow/cadence/pull/1880
* Optimize predeclared values, refactor runtime into reusable part by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1825
* [LS] Contract deploy test fix by [@sideninja](https://github.com/sideninja) in https://github.com/onflow/cadence/pull/1890
* Optimize interpreter configuration, reuse it in runtime environment by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1831
* Preallocate ordered maps by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1878
* Optimize import of crypto enum cases by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1888
* Append special prefix to all internal error string messages by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1901
* Improve hex decoding errors by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1904
* Move resource variables to shared state by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1902
* Reduce allocations by refactoring simple composite computed fields from map to function by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1899
* Refactor checker config and position info by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1898
* update BLS ciphersuite by [@tarakby](https://github.com/tarakby) in https://github.com/onflow/cadence/pull/1848
* [LS] Flow client encapsulation and integration refactor  by [@sideninja](https://github.com/sideninja) in https://github.com/onflow/cadence/pull/1752

### 📖 Documentation
* Add meeting notes for July 22, 2022 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1843
* Improve documentation for storage iteration by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1882
* Fix minor typo: language docs - consts & vars by [@sisyphusSmiling](https://github.com/sisyphusSmiling) in https://github.com/onflow/cadence/pull/1865

### Other Changes
* Fix batch script tool by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1861
* Parse version file and use max version by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1906
* Merging V0.25 branch back to master by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1879

## New Contributors
* [@sukantoraymond](https://github.com/sukantoraymond) made their first contribution in https://github.com/onflow/cadence/pull/1855
* [@sisyphusSmiling](https://github.com/sisyphusSmiling) made their first contribution in https://github.com/onflow/cadence/pull/1865
* [@bluesign](https://github.com/bluesign) made their first contribution in https://github.com/onflow/cadence/pull/1076

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.25.0...v0.26.0

[Changes][v0.26.0]


<a name="languageserver/v0.26.0"></a>
# [Language Server v0.26.0 (languageserver/v0.26.0)](https://github.com/onflow/cadence/releases/tag/languageserver/v0.26.0) - 12 Aug 2022

## 🛠 Improvements
This release introduces general improvements to the Language server code base that can be categorized into:
- **Isolating Flow client**
The Flow client was extracted and exposed via an interface, that allows us to mock it out during testing and to encapsulate all the flow client logic. 
- **Simplifying the state management**
There was a couple of complexities removed from the LS, most notably how accounts are managed, previously the accounts were managed on network, using the contract to resolve names to addresses, which provided another state that was required to be synced between LS, client and blockchain. This was removed and encapsulated within the Flow client implementation and it only contains a simple list of accounts which can be listed to the client via API if needed (clearly marking currently active account) and also can be easily switched between active accounts. No setup is required.
- **Better error handling**
Commands were refactored in how they handle errors, previously each command had to handle the error and send the response to the client, now the errors are returned from the command layer to the server where the response is handled in once place:
- **Testing**
The unit test of each of the integration components were added as well as end-to-end tests testing all the commands.
[#1752](https://github.com/onflow/cadence/issues/1752) [@sideninja](https://github.com/sideninja) 

[Changes][languageserver/v0.26.0]


<a name="v0.25.0"></a>
# [v0.25.0](https://github.com/onflow/cadence/releases/tag/v0.25.0) - 15 Aug 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.25.0 -->

## What's Changed

### ⭐ Features
* [LS] Add inlay hints by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1697
* Add AST inspector which allows efficient element traversal by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1730
* Show stack trace in runtime error by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1549
* Add dependency resolution package by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1634
* Contract analyzer by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1732
* Move unnecessary force lint check to the contract analyzer by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1743
* Migrate casting lints to the new linter by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1758
* Add defensive check for member accesses by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1759
* Migrate number conversion hints to the contract analyzer by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1763
* Add support for adding breakpoints to the debugger by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1736
* Expose runtime storage by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1679

### 🛠 Improvements
* Include static types when exporting arrays and dictionaries by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1762
* Switch to OpenTelemetry tracing library by [@SaveTheRbtz](https://github.com/SaveTheRbtz) in https://github.com/onflow/cadence/pull/1824
* Use pretty printing for AST element String implementations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1552
* [LS] Update types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1696
* Allow decoding of types in pre-0.3.0 format (just type ID) by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1734
* Suggest optional chaining when member access is on optional type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1614
* Improve pretty printing by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1520
* Improve batch script / contract requesting tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1619
* [LS] Run language server with Delve by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1629
* Tutorial Re-organization by [@joshuahannan](https://github.com/joshuahannan) in https://github.com/onflow/cadence/pull/1727
* Change script and transaction locations to arrays by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1735
* Improve analysis package by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1731
* Use location instead of location ID by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1742
* Pass interpreter instead of nil to StaticType function by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1754
* Bump onflow/atree from 0.3.0 to 0.4.0 by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/1760
* Prepare contract analyzer to be moved to separate repo by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1769
* Pass location range getter by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1756
* Remove contract analyzer package by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1771
* Separate user errors from internal errors by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1718
* Refactor parser errors by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1746
* Make `@onflow/cadence-parser` support node.js (Hybrid ESM/CJS) by [@Pure-Peace](https://github.com/Pure-Peace) in https://github.com/onflow/cadence/pull/1786
* Update language server to use new linter for hints by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1801
* Update go.js in NPM packages to Go 1.18.1 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1789
* Use generics for ordered map, lazily allocate inner map and list by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1804
* Don't copy code string if there's no error by [@pattyshack](https://github.com/pattyshack) in https://github.com/onflow/cadence/pull/1807
* Switch to generic list by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1808
* [LS] Use a hosted emulator from flowkit by [@sideninja](https://github.com/sideninja) in https://github.com/onflow/cadence/pull/1739
* refactor ExecuteTransaction body into Executor object by [@pattyshack](https://github.com/pattyshack) in https://github.com/onflow/cadence/pull/1797
* Refactor InvokeContractFunction into Executor object by [@pattyshack](https://github.com/pattyshack) in https://github.com/onflow/cadence/pull/1826
* refactor ExecuteScript body into Executor object by [@pattyshack](https://github.com/pattyshack) in https://github.com/onflow/cadence/pull/1800
* Refactor intervalst to be generic by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1833
* Fix typos and broken links on docs  by [@alilloig](https://github.com/alilloig) in https://github.com/onflow/cadence/pull/1834
* Fix tutorial numerations by [@alilloig](https://github.com/alilloig) in https://github.com/onflow/cadence/pull/1841
* Change Runtime.InvokeContractFunction signature by [@janezpodhostnik](https://github.com/janezpodhostnik) in https://github.com/onflow/cadence/pull/1811
* Replace `interface{}` with `any` by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1703
* Rename `parser2` package to `parser` by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1753
* Collape benchstat results by [@janezpodhostnik](https://github.com/janezpodhostnik) in https://github.com/onflow/cadence/pull/1782
* Language server should no longer depend on Flow CLI by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1788
* Separate type equality checker from contract update validator by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1787
* Append special prefix to all internal error string messages by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1858
* Refactor parser to always return syntax errors by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1764
* Update languageserver to newest version of flowkit by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1828

### 🐞 Bug Fixes
* Ignore missing comma in parameter list parse error for contract updates by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1712
* Fix decoding of repeated/recursive encoded types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1721
* Conversion to Word types should wrap instead of under/overflowing by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1723
* Fix deadlock and simplify debugger by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1728
* Fix capability check and borrow functions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1745
* Resource destruction should also invalidate references by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1744
* Forbid contract transfers by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1747
* Fail invalidated-resource use with a proper error by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1783
* Fix json encoding for builtin composite types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1792
* Panic on duplicate key for resource-typed dictionary literal by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1794
* Fix typo in how to run docgen tool by [@alilloig](https://github.com/alilloig) in https://github.com/onflow/cadence/pull/1810

### 📖 Documentation
* Update Crypto docs by [@masterEye-07](https://github.com/masterEye-07) in https://github.com/onflow/cadence/pull/1716
* update the space in value and types by [@danishtroon](https://github.com/danishtroon) in https://github.com/onflow/cadence/pull/1722
* Add documentation for type json encoding by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1725
* Improve type json docs by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1726
* Update design-patterns.mdx by [@masterEye-07](https://github.com/masterEye-07) in https://github.com/onflow/cadence/pull/1737
* Fix example code from the docs section "design patterns/Script-Accessible report" by [@fcostasilva](https://github.com/fcostasilva) in https://github.com/onflow/cadence/pull/1738
* Fixes tutorial callout and removes screenshot by [@joshuahannan](https://github.com/joshuahannan) in https://github.com/onflow/cadence/pull/1741
* fix hello world link and screenshot by [@joshuahannan](https://github.com/joshuahannan) in https://github.com/onflow/cadence/pull/1755
* Fix typo in nft-2 tutorial by [@ncpenke](https://github.com/ncpenke) in https://github.com/onflow/cadence/pull/1766
* Removes invalid MDX html comment by [@10thfloor](https://github.com/10thfloor) in https://github.com/onflow/cadence/pull/1780
* update the docs for design pattern by [@danishtroon](https://github.com/danishtroon) in https://github.com/onflow/cadence/pull/1776
* Update msg sender by [@saadtroon](https://github.com/saadtroon) in https://github.com/onflow/cadence/pull/1778
* Update values-and-types.mdx by [@masterEye-07](https://github.com/masterEye-07) in https://github.com/onflow/cadence/pull/1779
* Repair markdown for new site. by [@10thfloor](https://github.com/10thfloor) in https://github.com/onflow/cadence/pull/1795
* Update incorrect example code by [@jjiajun](https://github.com/jjiajun) in https://github.com/onflow/cadence/pull/1768
* Improve documentation by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1765
* Fix documentation, add linter by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1785
* Update 05-non-fungible-tokens-2.mdx by [@masterEye-07](https://github.com/masterEye-07) in https://github.com/onflow/cadence/pull/1820
* x has type Int8 by [@Skandesh](https://github.com/Skandesh) in https://github.com/onflow/cadence/pull/1859
* Update contract-updatability.md by [@ppichier](https://github.com/ppichier) in https://github.com/onflow/cadence/pull/1860
* Update anti-patterns.mdx by [@masterEye-07](https://github.com/masterEye-07) in https://github.com/onflow/cadence/pull/1798
* Update 05-non-fungible-tokens-2.mdx by [@masterEye-07](https://github.com/masterEye-07) in https://github.com/onflow/cadence/pull/1809
* Update capability-based-access-control.md by [@masterEye-07](https://github.com/masterEye-07) in https://github.com/onflow/cadence/pull/1814
* Update anti-patterns.mdx by [@masterEye-07](https://github.com/masterEye-07) in https://github.com/onflow/cadence/pull/1819

## New Contributors
* [@masterEye-07](https://github.com/masterEye-07) made their first contribution in https://github.com/onflow/cadence/pull/1716
* [@danishtroon](https://github.com/danishtroon) made their first contribution in https://github.com/onflow/cadence/pull/1722
* [@fcostasilva](https://github.com/fcostasilva) made their first contribution in https://github.com/onflow/cadence/pull/1738
* [@ncpenke](https://github.com/ncpenke) made their first contribution in https://github.com/onflow/cadence/pull/1766
* [@Pure-Peace](https://github.com/Pure-Peace) made their first contribution in https://github.com/onflow/cadence/pull/1786
* [@jjiajun](https://github.com/jjiajun) made their first contribution in https://github.com/onflow/cadence/pull/1768
* [@pattyshack](https://github.com/pattyshack) made their first contribution in https://github.com/onflow/cadence/pull/1807
* [@Skandesh](https://github.com/Skandesh) made their first contribution in https://github.com/onflow/cadence/pull/1859
* [@ppichier](https://github.com/ppichier) made their first contribution in https://github.com/onflow/cadence/pull/1860

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.24.1...v0.25.0

[Changes][v0.25.0]


<a name="languageserver/v0.25.0"></a>
# [Language Server v0.25.0 (languageserver/v0.25.0)](https://github.com/onflow/cadence/releases/tag/languageserver/v0.25.0) - 21 Jul 2022

## 🛠 Improvements
- Update languageserver to newest version of flowkit [#1828](https://github.com/onflow/cadence/issues/1828) [@dsainati1](https://github.com/dsainati1) 

## ⭐ Features
### Use a hosted emulator 
Language server now integrates and manages the emulator state. This makes LS clients simpler as they no longer have to take care of the blockchain execution and state sync.
[#1739](https://github.com/onflow/cadence/issues/1739) [@sideninja](https://github.com/sideninja) 


[Changes][languageserver/v0.25.0]


<a name="v0.24.6"></a>
# [v0.24.6](https://github.com/onflow/cadence/releases/tag/v0.24.6) - 24 Jun 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.24.6 -->

## What's Changed

### ⭐ Features

* Add defensive check for member accesses by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1761

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.24.5...v0.24.6

[Changes][v0.24.6]


<a name="v0.24.5"></a>
# [v0.24.5](https://github.com/onflow/cadence/releases/tag/v0.24.5) - 23 Jun 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.24.5 -->

## What's Changed

### 🐞 Bug Fixes

* Fix capability check and borrow functions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1748
* Resource destruction should also invalidate references by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1749
* Forbid contract transfers  by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1750


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.24.4...v0.24.5

[Changes][v0.24.5]


<a name="v0.24.4"></a>
# [v0.24.4](https://github.com/onflow/cadence/releases/tag/v0.24.4) - 17 Jun 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.24.4 -->

## What's Changed

### 🛠 Improvements
* Allow decoding of types in pre-0.3.0 format (just type ID) by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1733


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.24.3...v0.24.4

[Changes][v0.24.4]


<a name="v0.24.3"></a>
# [v0.24.3](https://github.com/onflow/cadence/releases/tag/v0.24.3) - 13 Jun 2022

<!-- Release notes generated using configuration in .github/release.yml at v0.24.3 -->

## What's Changed

### 🐞 Bug Fixes
* Fix decoding of repeated/recursive encoded types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1720


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.24.2...v0.24.3

[Changes][v0.24.3]


<a name="v0.24.2"></a>
# [v0.24.2](https://github.com/onflow/cadence/releases/tag/v0.24.2) - 10 Jun 2022

## What's Changed

### 🐞 Bug Fixes
* Ignore missing comma in parameter list parse error for contract updates by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1711

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.24.1...v0.24.2

[Changes][v0.24.2]


<a name="v0.24.1"></a>
# [v0.24.1](https://github.com/onflow/cadence/releases/tag/v0.24.1) - 09 Jun 2022

### 🐞 Bug Fixes

* Fix JSON encoding/decoding and test parallelization by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1709

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.24.0...v0.24.1

[Changes][v0.24.1]


<a name="languageserver/v0.24.0"></a>
# [Language Server v0.24.0 (languageserver/v0.24.0)](https://github.com/onflow/cadence/releases/tag/languageserver/v0.24.0) - 09 Jun 2022

<!-- Release notes generated using configuration in .github/release.yml at languageserver/v0.24.0 -->

## What's Changed

### 🛠 Improvements
* Update to Secure Cadence by [@sideninja](https://github.com/sideninja) in https://github.com/onflow/cadence/pull/1707

### 🐞 Bug Fixes
* Add stdlib builtin value declarations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1698

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.24.0...languageserver/v0.24.0

[Changes][languageserver/v0.24.0]


<a name="v0.24.0"></a>
# [v0.24 (Secure Cadence) (v0.24.0)](https://github.com/onflow/cadence/releases/tag/v0.24.0) - 07 Jun 2022

## What's Changed

The Secure Cadence milestone is focused on code-hardening and will enable the permissionless deployment of smart contracts on Mainnet. It is an important step in [our path to Stable Cadence](https://forum.onflow.org/t/the-path-to-stable-cadence/2702).

This required some breaking changes. You can read all about these changes and how to update your contracts in this forum post:
[Breaking changes coming with Secure Cadence release](https://forum.onflow.org/t/breaking-changes-coming-with-secure-cadence-release/3052)

While the focus was on securing Cadence, this release is also packed with many new features and improvements!

### 💥 Breaking Changes

* Zero-pad string representation of addresses by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1214
* Allow importing type and capability values by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1202
* Report an error when the parameter list is missing commas by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1073
* Use common super-type for expression type inferring by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1027
* `load`, `copy` and `borrow` cause a force cast  by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1286
* Taking a reference to an optional value now produces an optionally-typed reference by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1303
* Disallow arithmetic, comparison and bitwise operations on number supertypes by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1360
* Add CharacterValue type by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1392
* Limit mutation of container-typed fields to the scope of the composite by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1267

### ⭐ Features

* Allow runtime types to be used as a dictionary key by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1186
* Add constructor functions for run-time types by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1195
* Add new syntax to bind for-in loop indices  by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1216
* Find least common super-type by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1025
* Infer static type from the value itself for imported arrays/dictionaries by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1049
* Add a clone function to all values by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1229
* Allow scripts to access authorized accounts by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1240
* add `type(at: Path)` function for inspecting the type of objects in storage by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1254
* Report a hint for casting an expression of type `T` to same type `T` by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1072
* Add an interactive debugger to the command-line runner by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/940
* Start development of a new pretty printer for Cadence programs by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1024
* Pretty print statements by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1280
* Pretty print types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1281
* Add optional resource owner change tracking callback by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1289
* Add array slice function by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1315
* Add indexOf to array by [@bjartek](https://github.com/bjartek) in https://github.com/onflow/cadence/pull/1352
* Check transfer and use of transferred or destroyed resources at runtime by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1461
* Add a package to run a script against all accounts by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1434
* Memory metering by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1653
* Add a checker option to short-circuit error reporting by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1562

### 🛠 Improvements

* Add smoke tests cadence values / storage operations by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1185
* Clean up types in `cadence` package by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1204
* add missing types and conversion functions to cadence package by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1224
* Improve container mutation error message by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1228
* Updates Fungible Token tutorial to use a different structure and explanations by [@joshuahannan](https://github.com/joshuahannan) in https://github.com/onflow/cadence/pull/1238
* Re-apply "Remove receiver type from function type" by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1272
* Disable existing hints if linting is disabled by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1275
* Remove redundant subtyping check by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1285
* Tweak Cadence bench to be more stable by [@janezpodhostnik](https://github.com/janezpodhostnik) in https://github.com/onflow/cadence/pull/1290
* Improve state decoding tool by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1232
* Language Server updates by [@MaxStalker](https://github.com/MaxStalker) in https://github.com/onflow/cadence/pull/1295
* Support recursive supertype inferring for containers by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1319
* Add safety checks for go casts in the interpreter by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1331
* Ignore dead code by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1345
* Add safety checks for go casts in the parser by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1343
* Allows adding new interface conformances during a contract update by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1395
* Refactor type assertions of host function arguments by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1422
* Check use of invalidated resources at runtime by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1464
* Expand interpreter tracing for composite, array and dictionary types by [@ramtinms](https://github.com/ramtinms) in https://github.com/onflow/cadence/pull/1450
* Extend subtyping rule for restricted type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1486
* Check address length before decoding by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1522
* Add runtime subtype check using static types by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1487
* Improve new static type-based dynamic type checking by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1576
* Remove dynamic types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1577
* Flowkit version update by [@sideninja](https://github.com/sideninja) in https://github.com/onflow/cadence/pull/1534
* Adjust formulas for calculating memory usage of arithmetic operations on big integers by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1590
* Don't report errors for invalid types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1597
* [LS] Crash reporting by [@sideninja](https://github.com/sideninja) in https://github.com/onflow/cadence/pull/1600
* cadence.Value.Type method back to original param list by [@robert-e-davidson3](https://github.com/robert-e-davidson3) in https://github.com/onflow/cadence/pull/1633
* [LS] Debugging documentation and utility functions by [@sideninja](https://github.com/sideninja) in https://github.com/onflow/cadence/pull/1601
* Language reference docs improvement by [@alilloig](https://github.com/alilloig) in https://github.com/onflow/cadence/pull/1627
* Limit number of tokens which can be replayed (reparsed) by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1620
* Improve constant-sized type parsing by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1613
* [LS] NPM release v0.24.0 crash reporting by [@sideninja](https://github.com/sideninja) in https://github.com/onflow/cadence/pull/1641
* Refactor memory kinds and memory usages by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1667
* Improve error message for invalid public key by [@SaveTheRbtz](https://github.com/SaveTheRbtz) in https://github.com/onflow/cadence/pull/1665
* Limit depth of expressions and types by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1664
* Add token limit by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1668
* Fix local token replay limit, add global replay limit by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1663
* Handle and wrap non-error type panics in interpreterRuntime by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1678
* Update fxamacker/cbor and atree to latest versions by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/1682
* Differentiate invalidated resource use errors by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1706
* Optimize ByteSize functions for 7 Value types by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/1299
* Pool lexers to reduce allocations and improve performance by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1610
* Optimize HashInput functions to use scratch buffer provided by onflow/atree by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/1296
* Add more graceful handling for go type assertions in `runtime/sema` by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1318
* Gracefully handle casts during comparison operators by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1330

### 🐞 Bug Fixes

* Fix smoke tests by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1198
* Fix composite-typed argument passing by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1235
* Gracefully handle invalid restrictions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1249
* Replace `IsSubType` check with `IsSameTypeKind` check where necessary by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1032
* Gracefully handle casts during arithmetic operations by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1306
* Fix resource loss analysis and definite initialization analysis by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1332
* Fix and simplify parser buffering by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1333
* Fix type requirement checking by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1334
* Fix IsContainerType for CompositeType and InterfaceType by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1335
* Fix checking of `create before(...)` in post condition by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1336
* Fix type requirement checking by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1338
* Fix IsContainerType for CompositeType and InterfaceType by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1339
* Fix references to resource-kinded values by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1344
* Reject invalid access modifiers by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1347
* Fix parsing of 0_ integer literals by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1348
* Update uniseg to latest commit by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1370
* Fix definite field initialization analysis  by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1424
* Fix AST walking default-case in switch-statement by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1433
* Update atree by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1421
* Catch potential panics produced by resource owner changed callback by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1445
* Fix CI typo feature-secure-cadence by [@robert-e-davidson3](https://github.com/robert-e-davidson3) in https://github.com/onflow/cadence/pull/1465
* Fix resource field nesting check for enum raw value field by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1459
* do not wrap conditional in ${{}} by [@robert-e-davidson3](https://github.com/robert-e-davidson3) in https://github.com/onflow/cadence/pull/1473
* Fix field initialization: nested intersection should not override outer set by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1524
* Forbid reinitialization of variable, resource-kinded fields by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1527
* Commit storage before getting account storage capacity by [@robert-e-davidson3](https://github.com/robert-e-davidson3) in https://github.com/onflow/cadence/pull/1525
* Nil elements when reslicing by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1547
* Box optional-typed references to non-optional values by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1542
* Reads of missing storage maps should not cause creation by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1557
* Handle missing member when checking mutability of member by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1561
* [LS] Remove caching of checkers for imported programs by [@sideninja](https://github.com/sideninja) in https://github.com/onflow/cadence/pull/1572
* Properly transfer indexing expression by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1568
* fix double count memory by [@robert-e-davidson3](https://github.com/robert-e-davidson3) in https://github.com/onflow/cadence/pull/1593
* Fix common supertype calculation for optionals by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1594
* Fix erroneous report of resource invalidation when executing a pre-condition by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1625
* Revert removal of checker caching / [#1572](https://github.com/onflow/cadence/issues/1572) by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1628
* Track inner values of resource-kinded optionals by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1630
* Fix issue with programs cache on contract update by [@janezpodhostnik](https://github.com/janezpodhostnik) in https://github.com/onflow/cadence/pull/1644
* Fix parsing of reference expressions by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1622
* Wrap memory error in FatalError and check for it when parsing by [@robert-e-davidson3](https://github.com/robert-e-davidson3) in https://github.com/onflow/cadence/pull/1660
* Properly support super types in ConformsToStaticType implementations by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1694
* Fix JSON encoding of type values with recursive type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1705
* Avoid type-checking of the old program during contract updates by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1692


### 📖 Documentation

* Remove trailing comma in JSON cadence spec by [@fee1-dead](https://github.com/fee1-dead) in https://github.com/onflow/cadence/pull/1206
* Fix playground URL for this example by [@javiermanzano](https://github.com/javiermanzano) in https://github.com/onflow/cadence/pull/1209
* Fix comment typo by [@GuillaumeBAECHLER](https://github.com/GuillaumeBAECHLER) in https://github.com/onflow/cadence/pull/1210
* Fix typo on marketplace design code sample by [@javiermanzano](https://github.com/javiermanzano) in https://github.com/onflow/cadence/pull/1207
* Update ownerVault access control keyword by [@GuillaumeBAECHLER](https://github.com/GuillaumeBAECHLER) in https://github.com/onflow/cadence/pull/1212
* Fix grammar issue in Hello World tutorial by [@NessDan](https://github.com/NessDan) in https://github.com/onflow/cadence/pull/1231
* Add line break so images load by [@NessDan](https://github.com/NessDan) in https://github.com/onflow/cadence/pull/1242
* Change "Object" to plural (typo) by [@NessDan](https://github.com/NessDan) in https://github.com/onflow/cadence/pull/1241
* Update type inference docs by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1054
* Update 06-marketplace-compose.mdx by [@saadtroon](https://github.com/saadtroon) in https://github.com/onflow/cadence/pull/1265
* Include dependencies in contributing doc, plus gitignore. by [@robert-e-davidson3](https://github.com/robert-e-davidson3) in https://github.com/onflow/cadence/pull/1270
* Update and to are in 03-fungible-tokens.mdx by [@Nasir-Ali-Shah](https://github.com/Nasir-Ali-Shah) in https://github.com/onflow/cadence/pull/1271
* Update 06-marketplace-compose.mdx by [@saadtroon](https://github.com/saadtroon) in https://github.com/onflow/cadence/pull/1277
* update typo 0x03 to 0x02 by [@Nasir-Ali-Shah](https://github.com/Nasir-Ali-Shah) in https://github.com/onflow/cadence/pull/1278
* Updates NFT tutorial to use latest best practices by [@joshuahannan](https://github.com/joshuahannan) in https://github.com/onflow/cadence/pull/1301
* Start of a symbol/operator glossary by [@muttoni](https://github.com/muttoni) in https://github.com/onflow/cadence/pull/1279
* Move resource owner section to resources page by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1346
* Fixes glossary page by [@muttoni](https://github.com/muttoni) in https://github.com/onflow/cadence/pull/1350
* Fix links in callouts, linting by [@alxflw](https://github.com/alxflw) in https://github.com/onflow/cadence/pull/1359
* Update 02-hello-world.mdx by [@ObjectPlayer](https://github.com/ObjectPlayer) in https://github.com/onflow/cadence/pull/1365
* Fix bad links in doc by [@kerrywei](https://github.com/kerrywei) in https://github.com/onflow/cadence/pull/1372
* Update 02-hello-world.mdx by [@elpunkt](https://github.com/elpunkt) in https://github.com/onflow/cadence/pull/1357
* update tutorial links for new playground format by [@joshuahannan](https://github.com/joshuahannan) in https://github.com/onflow/cadence/pull/1387
* Object player/1358 fix broken links in cadence document by [@ObjectPlayer](https://github.com/ObjectPlayer) in https://github.com/onflow/cadence/pull/1368
* Update resource name by [@brunogonzales](https://github.com/brunogonzales) in https://github.com/onflow/cadence/pull/1390
* Update Hello World tutorial w/ Video Guide by [@kimcodeashian](https://github.com/kimcodeashian) in https://github.com/onflow/cadence/pull/1401
* Add example of a script to add a key to existing account by [@justinbarry](https://github.com/justinbarry) in https://github.com/onflow/cadence/pull/1403
* Embedding Fungible Token video guide + description by [@kimcodeashian](https://github.com/kimcodeashian) in https://github.com/onflow/cadence/pull/1414
* Updating Fungible Token video to correct source by [@kimcodeashian](https://github.com/kimcodeashian) in https://github.com/onflow/cadence/pull/1418
* Fix comments for StorageMap ReadValue and WriteValue by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1439
* Marketplace tutorial improvements by [@joshuahannan](https://github.com/joshuahannan) in https://github.com/onflow/cadence/pull/1412
* Capability-based Access Control Document Improvements by [@kgebreki](https://github.com/kgebreki) in https://github.com/onflow/cadence/pull/1519
* Update the crypto doc by [@tarakby](https://github.com/tarakby) in https://github.com/onflow/cadence/pull/1548
* Add `type` function to list of functions in `AuthAccount` documentation by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1676
* Best Practices and Anti-Patterns Updates by [@joshuahannan](https://github.com/joshuahannan) in https://github.com/onflow/cadence/pull/1471
* Document number supertype changes by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1702
* Update documentation for optional references by [@dsainati1](https://github.com/dsainati1) https://github.com/onflow/cadence/pull/1700
* Document new short address toString behaviour by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1699
* Fixes typo in tutorial by [@walski](https://github.com/walski) in https://github.com/onflow/cadence/pull/1302
* Update Resource documentation link in 02-hello-world.mdx  by [@gnujoow](https://github.com/gnujoow) in https://github.com/onflow/cadence/pull/1298
* Update 02-hello-world.mdx by [@gnujoow](https://github.com/gnujoow) in https://github.com/onflow/cadence/pull/1297
* Update docs/design-patterns backquote paths by [@KeisukeYamashita](https://github.com/KeisukeYamashita) in https://github.com/onflow/cadence/pull/1323

### Other Changes

* Add smoke test to use only enum (CompositeValue) as dictionary key by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/1201
* Update golangci-lint by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1200
* Add container mutation tests for inserting functions into dictionaries by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1215
* Add benchstat GH action by [@janezpodhostnik](https://github.com/janezpodhostnik) in https://github.com/onflow/cadence/pull/1227
* Add test for casting storage ref to an invalid type by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1243
* Language server: Add test for diagnostics by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1233
* Update LS dependencies by [@sideninja](https://github.com/sideninja) in https://github.com/onflow/cadence/pull/1163
* Merge v0.20.x changes by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1292
* Fix npm packages by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1293
* Update testify by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1274
* Update to Go 1.16 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1361
* Export `StringAtreeValue` from the interpreter by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1366
* Panic when testRuntimeInterface methods need hooks but lack them by [@robert-e-davidson3](https://github.com/robert-e-davidson3) in https://github.com/onflow/cadence/pull/1399
* Add tests for linking to already linked paths by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1416
* Add Daniel as code owner by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1436
* Improve bug report issue template by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1440
* use feature branch for cadence->flow-go sync PRs by [@robert-e-davidson3](https://github.com/robert-e-davidson3) in https://github.com/onflow/cadence/pull/1441
* Add test for CharacterValue.HashInput by [@robert-e-davidson3](https://github.com/robert-e-davidson3) in https://github.com/onflow/cadence/pull/1432
* Various regression tests by [@robert-e-davidson3](https://github.com/robert-e-davidson3) in https://github.com/onflow/cadence/pull/1425
* Check capabilities of dependencies by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1476
* Add a custom linter which detects Value composite literals by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1481
* Bump codecov-action from v1 to v2 in ci.yml by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/1492
* Fix lint errors by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1509
* Update copyright by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1529
* Added test to partially ensure HashableValue does not regress by [@robert-e-davidson3](https://github.com/robert-e-davidson3) in https://github.com/onflow/cadence/pull/1431
* Remove release drafter GitHub action by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1551
* Add codecov config by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1553
* Add tests for casting references inside containers by [@SupunS](https://github.com/SupunS) in https://github.com/onflow/cadence/pull/1581
* Add fast-test option to makefile by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1589
* [LS] Update version by [@sideninja](https://github.com/sideninja) in https://github.com/onflow/cadence/pull/1583
* Disable flow-go sync by [@janezpodhostnik](https://github.com/janezpodhostnik) in https://github.com/onflow/cadence/pull/1645
* Update to Go 1.18 by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1621

## New Contributors

We would like to thank all of our new contributors improving Cadence:

* [@fee1-dead](https://github.com/fee1-dead) made their first contribution in https://github.com/onflow/cadence/pull/1206
* [@javiermanzano](https://github.com/javiermanzano) made their first contribution in https://github.com/onflow/cadence/pull/1209
* [@GuillaumeBAECHLER](https://github.com/GuillaumeBAECHLER) made their first contribution in https://github.com/onflow/cadence/pull/1210
* [@NessDan](https://github.com/NessDan) made their first contribution in https://github.com/onflow/cadence/pull/1231
* [@saadtroon](https://github.com/saadtroon) made their first contribution in https://github.com/onflow/cadence/pull/1265
* [@Nasir-Ali-Shah](https://github.com/Nasir-Ali-Shah) made their first contribution in https://github.com/onflow/cadence/pull/1271
* [@walski](https://github.com/walski) made their first contribution in https://github.com/onflow/cadence/pull/1302
* [@gnujoow](https://github.com/gnujoow) made their first contribution in https://github.com/onflow/cadence/pull/1298
* [@muttoni](https://github.com/muttoni) made their first contribution in https://github.com/onflow/cadence/pull/1279
* [@KeisukeYamashita](https://github.com/KeisukeYamashita) made their first contribution in https://github.com/onflow/cadence/pull/1323
* [@alxflw](https://github.com/alxflw) made their first contribution in https://github.com/onflow/cadence/pull/1359
* [@ObjectPlayer](https://github.com/ObjectPlayer) made their first contribution in https://github.com/onflow/cadence/pull/1365
* [@elpunkt](https://github.com/elpunkt) made their first contribution in https://github.com/onflow/cadence/pull/1357
* [@brunogonzales](https://github.com/brunogonzales) made their first contribution in https://github.com/onflow/cadence/pull/1390
* [@justinbarry](https://github.com/justinbarry) made their first contribution in https://github.com/onflow/cadence/pull/1403
* [@kgebreki](https://github.com/kgebreki) made their first contribution in https://github.com/onflow/cadence/pull/1519
* [@alilloig](https://github.com/alilloig) made their first contribution in https://github.com/onflow/cadence/pull/1627
* [@SaveTheRbtz](https://github.com/SaveTheRbtz) made their first contribution in https://github.com/onflow/cadence/pull/1665

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.23.0...v0.24.0

[Changes][v0.24.0]


<a name="languageserver/v0.23.4"></a>
# [Language Server v0.23.4 (languageserver/v0.23.4)](https://github.com/onflow/cadence/releases/tag/languageserver/v0.23.4) - 26 Apr 2022

## 🛠 Improvements
- Add the Sentry crash reporting option if enabled by the client [#1600](https://github.com/onflow/cadence/issues/1600) [@sideninja](https://github.com/sideninja) 

[Changes][languageserver/v0.23.4]


<a name="languageserver/v0.23.0"></a>
# [Language Server v0.23.0 (languageserver/v0.23.0)](https://github.com/onflow/cadence/releases/tag/languageserver/v0.23.0) - 13 Apr 2022

## 🐞 Bug Fixes
**Diagnostics not working on imported programs changes**
When importing a program was changed it didn't re-check it leading to outdated diagnostics provided to the client. This was fixed by always re-checking imported programs. [#1572](https://github.com/onflow/cadence/issues/1572) [@sideninja](https://github.com/sideninja) 

[Changes][languageserver/v0.23.0]


<a name="v0.23.4"></a>
# [v0.23.4](https://github.com/onflow/cadence/releases/tag/v0.23.4) - 12 Apr 2022

## 🐞 Bug Fixes

* Transfer indexing expression by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1565

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.23.3...v0.23.4

[Changes][v0.23.4]


<a name="v0.23.3"></a>
# [v0.23.3](https://github.com/onflow/cadence/releases/tag/v0.23.3) - 01 Apr 2022

## 🐞 Bug Fixes

* Fix overflow error in Cadence `UInt` to Go `int` conversion by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1550

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.23.2...v0.23.3

[Changes][v0.23.3]


<a name="v0.23.2"></a>
# [v0.23.2](https://github.com/onflow/cadence/releases/tag/v0.23.2) - 11 Mar 2022

## 🛠 Improvements

* Move computation metering and limit enforcement responsibility to the environment by [@ramtinms](https://github.com/ramtinms), [@janezpodhostnik](https://github.com/janezpodhostnik) in https://github.com/onflow/cadence/pull/1469, https://github.com/onflow/cadence/pull/1496


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.23.1...v0.23.2

[Changes][v0.23.2]


<a name="v0.23.1"></a>
# [v0.23.1](https://github.com/onflow/cadence/releases/tag/v0.23.1) - 07 Mar 2022

## 🛠 Improvements

* Extend subtyping rule for restricted type by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1480

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.23.0...v0.23.1

[Changes][v0.23.1]


<a name="v0.23.0"></a>
# [v0.23.0](https://github.com/onflow/cadence/releases/tag/v0.23.0) - 03 Mar 2022

## 💥 Breaking Changes

* Remove `PublicKey.isValid` by [@robert-e-davidson3](https://github.com/robert-e-davidson3) in https://github.com/onflow/cadence/pull/1273, https://github.com/onflow/cadence/pull/1454

  The `PublicKey` constructor now validates the public key data and aborts the program if it is invalid. 
  The `isValid` field was removed.

## ⭐ Features

* Add converter functions to produce path values from strings by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1437, https://github.com/onflow/cadence/pull/1211

  Paths can now be constructed at run-time. The following path constructors were added:

  ```kotlin
  fun PublicPath(identifier: string): PublicPath?
  fun PrivatePath(identifier: string): PrivatePath?
  fun StoragePath(identifier: string): StoragePath?
  ```

  Each of these functions take an identifier and produce a path of the appropriate domain.
  For example, to construct a public path from a string:
  
  ```kotlin
  let pathID = "foo"
  let path = PublicPath(identifier: pathID) // is /public/foo
  ```

  For more information, see https://docs.onflow.org/cadence/language/accounts/#path-functions

* Add RLP decoding utility methods by [@ramtinms](https://github.com/ramtinms) in https://github.com/onflow/cadence/issues/1397, https://github.com/onflow/cadence/pull/1456

  Cadence now provides RLP decoding functions through the built-in contract `RLP`.
  It provides the following functions for decoding strings and lists:

  ```kotlin
  fun decodeString(_ input: [UInt8]): [UInt8]
  fun decodeList(_ input: [UInt8]): [[UInt8]]
  ```

  For more information, see https://docs.onflow.org/cadence/language/built-in-functions/#rlp

* Add index to for loop by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1216, https://github.com/onflow/cadence/pull/1452

  For-loops may now declare an indexing variable:

  ```kotlin
  for index, element in array {
      // ...
  }
  ```

* Add BLS-specific crypto functions by [@dsainati1](https://github.com/dsainati1) in https://github.com/onflow/cadence/pull/1236, https://github.com/onflow/cadence/pull/1250, https://github.com/onflow/cadence/pull/1430

  `PublicKey` gained a new function to verify a proof-of-possession:
  
  ```kotlin
  fun verifyPoP(_ proof: [UInt8]): Bool
  ```

  Cadence now provides BLS-specific functions through the built-in contract `BLS`. 
  It provides the following functions:

  ```kotlin
  fun aggregateSignatures(_ signatures: [[UInt8]]): [UInt8]?
  fun aggregatePublicKeys(_ publicKeys: [PublicKey]): PublicKey?
  ```

  For more information, see https://docs.onflow.org/cadence/language/crypto/#bls

* Adding tracing by [@ramtinms](https://github.com/ramtinms) in https://github.com/onflow/cadence/pull/1179, https://github.com/onflow/cadence/pull/1448

## 🛠 Improvements

* Refactor account storage to use atree ordered maps by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1248, https://github.com/onflow/cadence/pull/1404 https://github.com/onflow/cadence/pull/1391

  This storage format change will allow [adding a storage querying API](https://github.com/onflow/cadence/issues/208) in the future.

* Expand interpreter tracing for composite, array and dictionary types by [@ramtinms](https://github.com/ramtinms) in https://github.com/onflow/cadence/pull/1450, https://github.com/onflow/cadence/pull/1468
* Bump onflow/atree to v0.2.0 by [@fxamacker](https://github.com/fxamacker) in https://github.com/onflow/cadence/pull/1470

  This updates Cadence to the [latest version of atree](https://github.com/onflow/atree/releases/tag/v0.2.0).

* Improve BLS API by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1417
* Improve RLP API by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1423

## 🐞 Bug Fixes

* Make error reporting in contract update validation deterministic by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1420, https://github.com/onflow/cadence/pull/1428

## 🙌 New Contributors

Thank you to our new contributor, [@robert-e-davidson3](https://github.com/robert-e-davidson3)!

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.21.3...v0.23.0

[Changes][v0.23.0]


<a name="v0.21.3"></a>
# [v0.21.3](https://github.com/onflow/cadence/releases/tag/v0.21.3) - 01 Mar 2022

## 🐞 Bug Fixes

* Fix resource field nesting check for enum raw value field by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1462
* Fix resource checking by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1463

**Full Changelog**: https://github.com/onflow/cadence/compare/v0.21.2...v0.21.3

[Changes][v0.21.3]


<a name="v0.21.2"></a>
# [v0.21.2](https://github.com/onflow/cadence/releases/tag/v0.21.2) - 16 Feb 2022

## 🐞 Bug Fixes

* Require static types for all host function values by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1407
* Fix boxing in casts by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1408
* Fix EOF emission in lexer by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1409
* Consider halting when determining definite invalidation by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1411
* Fix acceptance of buffered errors by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1410
* Make error reporting in contract update validation deterministic by [@turbolent](https://github.com/turbolent) in https://github.com/onflow/cadence/pull/1427


**Full Changelog**: https://github.com/onflow/cadence/compare/v0.21.1...v0.21.2

[Changes][v0.21.2]


<a name="v0.21.1"></a>
# [v0.21.1](https://github.com/onflow/cadence/releases/tag/v0.21.1) - 03 Feb 2022

## 🛠 Improvements

* Allow adding new interface conformances during a contract update ([#1396](https://github.com/onflow/cadence/issues/1396)) [@SupunS](https://github.com/SupunS) 

[Changes][v0.21.1]


<a name="v0.21.0"></a>
# [v0.21.0](https://github.com/onflow/cadence/releases/tag/v0.21.0) - 26 Jan 2022

## ⭐ Features

- Add Keccak256 hash to Cadence ([#1327](https://github.com/onflow/cadence/issues/1327)) [@zhiqiangxu](https://github.com/zhiqiangxu)

## 🐞 Bug Fixes

- Gracefully handle Cadence crashes/panics ([#1364](https://github.com/onflow/cadence/issues/1364)) [@SupunS](https://github.com/SupunS)
- Ownership of nested resources is inconsistent, breaks backwards compatibility ([#1320](https://github.com/onflow/cadence/issues/1320)) [@turbolent](https://github.com/turbolent) 
- Fix resource loss analysis and definite initialization analysis ([#1332](https://github.com/onflow/cadence/issues/1332)) [@turbolent](https://github.com/turbolent) 
- Fix integer conversion ([#1208](https://github.com/onflow/cadence/issues/1208)) [@turbolent](https://github.com/turbolent) 
- Always check address overflow ([#1349](https://github.com/onflow/cadence/issues/1349)) [@turbolent](https://github.com/turbolent) 

[Changes][v0.21.0]


<a name="v0.20.0"></a>
# [v0.20.0](https://github.com/onflow/cadence/releases/tag/v0.20.0) - 17 Nov 2021

This release culminates a half-year effort to improve the Cadence storage layer, i.e. how Cadence values are stored in accounts, in particular arrays, dictionaries, and composite values. [@fxamacker](https://github.com/fxamacker) and [@ramtinms](https://github.com/ramtinms) designed and implemented completely new array and ordered map data structures, resulting in [atree](https://github.com/onflow/atree). These data structures are fast and scalable: they allow efficient access and allow efficient modifications to arrays and dictionaries with very large number of elements. [@turbolent](https://github.com/turbolent) and [@SupunS](https://github.com/SupunS) integrated this new library into Cadence. This new storage layer also fixed a bug in the resource `owner` field. Please see the breaking changes section for more details.

In addition, this release saw several feature and documentation contributions by new community members. Thank you!

## 💥 Breaking Changes

- With the storage layer update, the `owner` field of resources is now implemented correctly, as originally designed:
  The field is only non-`nil` if the resource is currently in storage, and is always `nil` if the resource is on the stack.
  This breaking change is actually a bug fix, as the `owner` field was previously implemented incorrectly: After a resource was moved out of storage onto the stack, the `owner` field was still referring to the account the resource was stored in.

  Please ensure your Cadence programs do not rely on the broken behaviour and update them if needed.

  For example, in a function that is passed a resource as an argument, the resource's `owner` field will always be `nil`:

  ```kotlin
  pub fun deposit(token: @NonFungibleToken.NFT) {
      // token.owner is always nil!
      // ...
  }
  ```

- Add limit on call stack depth ([#1138](https://github.com/onflow/cadence/issues/1138)) [@SupunS](https://github.com/SupunS)

## ⭐ Features

- Add `Path.toString()` with tests and documentation. ([#1141](https://github.com/onflow/cadence/issues/1141)) [@rheaplex](https://github.com/rheaplex)
- Adding `toLower()` to `String` type ([#1176](https://github.com/onflow/cadence/issues/1176)) [@bjartek](https://github.com/bjartek)
- Add `isSubtype` function for runtime values ([#1189](https://github.com/onflow/cadence/issues/1189)) [@dsainati1](https://github.com/dsainati1)

## 🛠 Improvements

- Introduce `TokenStream` interface ([#1113](https://github.com/onflow/cadence/issues/1113)) [@jsatdapr](https://github.com/jsatdapr)
- Remove obsolete script/transaction parameter check ([#1114](https://github.com/onflow/cadence/issues/1114)) [@SupunS](https://github.com/SupunS) 
- Add static type to host functions ([#1144](https://github.com/onflow/cadence/issues/1144)) [@SupunS](https://github.com/SupunS)
- Infer RHS of nil-coalescing operator to have non-optional type of LHS ([#1154](https://github.com/onflow/cadence/issues/1154)) [@turbolent](https://github.com/turbolent)
- Use new atree library for arrays, dictionaries, and composite values ([#1156](https://github.com/onflow/cadence/issues/1156)) [@turbolent](https://github.com/turbolent)
- Limit error line lengths ([#1165](https://github.com/onflow/cadence/issues/1165)) [@turbolent](https://github.com/turbolent)
- Add dynamic sub-typing for function types ([#1172](https://github.com/onflow/cadence/issues/1172)) [@SupunS](https://github.com/SupunS)
- Remove `CompositeDeclarations` check from `FunctionEntryPointDeclaration` ([#1175](https://github.com/onflow/cadence/issues/1175)) [@janezpodhostnik](https://github.com/janezpodhostnik)
- Check slab and account storage health ([#1193](https://github.com/onflow/cadence/issues/1193)) [@turbolent](https://github.com/turbolent)

## 🐞 Bug Fixes

- Fix export errors by properly wrapping in runtime errors ([#1139](https://github.com/onflow/cadence/issues/1139)) [@SupunS](https://github.com/SupunS)
- Fix array function member type creation ([#1146](https://github.com/onflow/cadence/issues/1146)) [@SupunS](https://github.com/SupunS)
- Add missing function type for `Path.toString` ([#1150](https://github.com/onflow/cadence/issues/1150)) [@turbolent](https://github.com/turbolent)
- Consider constant-sized type's size when checking dynamic array type ([#1158](https://github.com/onflow/cadence/issues/1158)) [@turbolent](https://github.com/turbolent)
- Fix move operator on empty resource collections ([#1159](https://github.com/onflow/cadence/issues/1159)) [@SupunS](https://github.com/SupunS)
- Fix block ID static type ([#1168](https://github.com/onflow/cadence/issues/1168)) [@turbolent](https://github.com/turbolent)
- Reintroduce and fix `stringAtreeValue` ([#1174](https://github.com/onflow/cadence/issues/1174)) [@turbolent](https://github.com/turbolent)
- Fix reference uses after moves ([#1180](https://github.com/onflow/cadence/issues/1180)) [@turbolent](https://github.com/turbolent)

## 📖 Documentation

- Core events documentation ([#1110](https://github.com/onflow/cadence/issues/1110)) [@sideninja](https://github.com/sideninja)
- Migrate remaining docs from docs site. ([#1155](https://github.com/onflow/cadence/issues/1155)) [@10thfloor](https://github.com/10thfloor)
- Fix bad link in Cadence tutorial page 01 ([#1167](https://github.com/onflow/cadence/issues/1167)) [@kerrywei](https://github.com/kerrywei)
- Fixing typo ([#1182](https://github.com/onflow/cadence/issues/1182)) [@sovietspy2](https://github.com/sovietspy2)
- Fix units on storageUsed and storageCapacity data members (bytes) ([#1184](https://github.com/onflow/cadence/issues/1184)) [@Giners](https://github.com/Giners)
- Fix typos ([#1187](https://github.com/onflow/cadence/issues/1187), [#1188](https://github.com/onflow/cadence/issues/1188), [#1191](https://github.com/onflow/cadence/issues/1191), [#1194](https://github.com/onflow/cadence/issues/1194)) [@markstrefford](https://github.com/markstrefford)

## 🙌 New Contributors

Thank you to all new contributors:

* [@rheaplex](https://github.com/rheaplex) 
* [@dsainati1](https://github.com/dsainati1)
* [@jsatdapr](https://github.com/jsatdapr) 
* [@kerrywei](https://github.com/kerrywei) 
* [@sovietspy2](https://github.com/sovietspy2) 
* [@markstrefford](https://github.com/markstrefford) 
* [@Giners](https://github.com/Giners) 

## Changelog

https://github.com/onflow/cadence/compare/v0.19.1...v0.20.0

[Changes][v0.20.0]


<a name="v0.19.1"></a>
# [v0.19.1](https://github.com/onflow/cadence/releases/tag/v0.19.1) - 13 Sep 2021

## 🛠 Improvements

- Remove obsolete script/transaction parameter check ([#1114](https://github.com/onflow/cadence/issues/1114)) [@SupunS](https://github.com/SupunS) 

## 🐞 Bug Fixes

- Fix export errors by properly wrapping in runtime errors ([#1139](https://github.com/onflow/cadence/issues/1139)) [@SupunS](https://github.com/SupunS)

[Changes][v0.19.1]


<a name="v0.19.0"></a>
# [v0.19.0](https://github.com/onflow/cadence/releases/tag/v0.19.0) - 13 Sep 2021

## 💥 Breaking Changes

- Fix export of values ([#1067](https://github.com/onflow/cadence/issues/1067)) [@turbolent](https://github.com/turbolent)
- Add encoding/decoding for array value static type ([#1035](https://github.com/onflow/cadence/issues/1035)) [@SupunS](https://github.com/SupunS)

## ⭐ Features

- Add public account contracts ([#1090](https://github.com/onflow/cadence/issues/1090)) [@SupunS](https://github.com/SupunS)
- Add `names` field to auth account contracts ([#1089](https://github.com/onflow/cadence/issues/1089)) [@SupunS](https://github.com/SupunS)
- Report a hint when casting to a same type as the expected type ([#1056](https://github.com/onflow/cadence/issues/1056)) [@SupunS](https://github.com/SupunS)
- Add encoding/decoding for array value static type ([#1035](https://github.com/onflow/cadence/issues/1035)) [@SupunS](https://github.com/SupunS)
- Add walker for values ([#1037](https://github.com/onflow/cadence/issues/1037)) [@turbolent](https://github.com/turbolent)
- Add encoding/decoding dictionary static type info ([#1036](https://github.com/onflow/cadence/issues/1036)) [@SupunS](https://github.com/SupunS)
- Store static type in array/dictionary values ([#1041](https://github.com/onflow/cadence/issues/1041)) [@SupunS](https://github.com/SupunS)
- Infer static-type from imported array/dictionary values ([#1052](https://github.com/onflow/cadence/issues/1052)) [@SupunS](https://github.com/SupunS)

## 🛠 Improvements

- Add type inferring for func-args when there are no generics ([#1033](https://github.com/onflow/cadence/issues/1033)) [@SupunS](https://github.com/SupunS)
- Add container static types ([#1125](https://github.com/onflow/cadence/issues/1125)) [@SupunS](https://github.com/SupunS)
- Add container mutation check for arrays and dictionaries ([#1103](https://github.com/onflow/cadence/issues/1103)) [@SupunS](https://github.com/SupunS)
- flow-go sync tweaks ([#1104](https://github.com/onflow/cadence/issues/1104)) [@janezpodhostnik](https://github.com/janezpodhostnik)
- Return a dedicated error when decoding fails due to an unsupported CBOR tag ([#1064](https://github.com/onflow/cadence/issues/1064)) [@turbolent](https://github.com/turbolent)
- Remove obsolete fields ([#1053](https://github.com/onflow/cadence/issues/1053)) [@turbolent](https://github.com/turbolent)
- Include and consider static types of arrays and dictionaries ([#1043](https://github.com/onflow/cadence/issues/1043)) [@turbolent](https://github.com/turbolent)
- Infer array and dictionary static types from expected type during import ([#1038](https://github.com/onflow/cadence/issues/1038)) [@turbolent](https://github.com/turbolent)
- Finish storage format changes ([#1042](https://github.com/onflow/cadence/issues/1042)) [@turbolent](https://github.com/turbolent)
- Add static types to array values and dictionary values ([#1034](https://github.com/onflow/cadence/issues/1034)) [@turbolent](https://github.com/turbolent)
- FlowKit API Update ([#1028](https://github.com/onflow/cadence/issues/1028)) [@sideninja](https://github.com/sideninja)
- Produce valid Cadence string literals when formatting strings ([#1023](https://github.com/onflow/cadence/issues/1023)) [@turbolent](https://github.com/turbolent)
- Directly declare base values in activation without interpreter ([#1022](https://github.com/onflow/cadence/issues/1022)) [@turbolent](https://github.com/turbolent)
- Reuse base activation across all interpreters ([#1018](https://github.com/onflow/cadence/issues/1018)) [@SupunS](https://github.com/SupunS)
- Extend the source compatibility suite ([#980](https://github.com/onflow/cadence/issues/980)) [@turbolent](https://github.com/turbolent)
- Update the language server to the latest Cadence version ([#1015](https://github.com/onflow/cadence/issues/1015)) [@turbolent](https://github.com/turbolent)
- Convert the maps in the type file to simple arrays ([#1086](https://github.com/onflow/cadence/issues/1086)) [@jwinkler2083233](https://github.com/jwinkler2083233)
- Update Security to point to website ([#1074](https://github.com/onflow/cadence/issues/1074)) [@jkan2](https://github.com/jkan2)
- Gracefully handle the optional-chaining invocation on non-optional member ([#1071](https://github.com/onflow/cadence/issues/1071)) [@turbolent](https://github.com/turbolent)
- Add static type sanity check for imported values ([#1048](https://github.com/onflow/cadence/issues/1048)) [@SupunS](https://github.com/SupunS)
- Add tests for decoding from old format and encoding in new format ([#1046](https://github.com/onflow/cadence/issues/1046)) [@SupunS](https://github.com/SupunS)
- Add more tests for imported array/dictionary value type conformance ([#1045](https://github.com/onflow/cadence/issues/1045)) [@SupunS](https://github.com/SupunS)
- Add encoder/decoder v5 ([#1039](https://github.com/onflow/cadence/issues/1039)) [@SupunS](https://github.com/SupunS)
- Remove additional check for builtin values during import resolving ([#1026](https://github.com/onflow/cadence/issues/1026)) [@SupunS](https://github.com/SupunS)
- Update NPM packages ([#1011](https://github.com/onflow/cadence/issues/1011)) [@turbolent](https://github.com/turbolent)

## 🐞 Bug Fixes

- Cointainer variances fixes ([#1087](https://github.com/onflow/cadence/issues/1087)) [@turbolent](https://github.com/turbolent)
- Check member reads and writes ([#1085](https://github.com/onflow/cadence/issues/1085)) [@turbolent](https://github.com/turbolent)
- Include receiver in function type and check it in invocations  ([#1084](https://github.com/onflow/cadence/issues/1084)) [@turbolent](https://github.com/turbolent)
- Fix race condition in hash and sign algorithm values ([#1096](https://github.com/onflow/cadence/issues/1096)) [@SupunS](https://github.com/SupunS)
- Check run-time type of the argument of getCapability calls ([#1083](https://github.com/onflow/cadence/issues/1083)) [@turbolent](https://github.com/turbolent)
- Fix interpretation of optional binding with second value ([#1082](https://github.com/onflow/cadence/issues/1082)) [@turbolent](https://github.com/turbolent)
- Check resource construction ([#1081](https://github.com/onflow/cadence/issues/1081)) [@turbolent](https://github.com/turbolent)
- Fix address conversion ([#1080](https://github.com/onflow/cadence/issues/1080)) [@turbolent](https://github.com/turbolent)
- Gracefully handle errors during transaction and script argument validation ([#1070](https://github.com/onflow/cadence/issues/1070)) [@turbolent](https://github.com/turbolent)
- Fix map keys ([#1069](https://github.com/onflow/cadence/issues/1069)) [@turbolent](https://github.com/turbolent)
- Fix import of values ([#1068](https://github.com/onflow/cadence/issues/1068)) [@turbolent](https://github.com/turbolent)
- Fix export of values ([#1067](https://github.com/onflow/cadence/issues/1067)) [@turbolent](https://github.com/turbolent)
- Fix get and revoke key functions ([#1065](https://github.com/onflow/cadence/issues/1065)) [@turbolent](https://github.com/turbolent)
- Finish storage format changes ([#1042](https://github.com/onflow/cadence/issues/1042)) [@turbolent](https://github.com/turbolent)
- Fix close-brace for struct definition docs in docgen tool ([#1019](https://github.com/onflow/cadence/issues/1019)) [@SupunS](https://github.com/SupunS)
- Don't check missing program ([#1013](https://github.com/onflow/cadence/issues/1013)) [@turbolent](https://github.com/turbolent)

## 📖 Documentation

- Give Resources their own page. ([#1130](https://github.com/onflow/cadence/issues/1130)) [@10thfloor](https://github.com/10thfloor)
- Fix and improve documentation ([#1122](https://github.com/onflow/cadence/issues/1122)) [@turbolent](https://github.com/turbolent)
- Add presentation "Programming Language Implementation / Cadence Implementation" ([#990](https://github.com/onflow/cadence/issues/990)) [@turbolent](https://github.com/turbolent)
- Remove Crypto contract status callouts ([#1012](https://github.com/onflow/cadence/issues/1012)) [@turbolent](https://github.com/turbolent)
- Improve the documentation for dictionaries and arrays ([#1020](https://github.com/onflow/cadence/issues/1020)) [@turbolent](https://github.com/turbolent)
- Fix operator documentation ([#1017](https://github.com/onflow/cadence/issues/1017)) [@turbolent](https://github.com/turbolent)
- Updating broken links in ReadMe ([#1097](https://github.com/onflow/cadence/issues/1097)) [@kimcodeashian](https://github.com/kimcodeashian)

[Changes][v0.19.0]


<a name="v0.18.0"></a>
# [v0.18.0](https://github.com/onflow/cadence/releases/tag/v0.18.0) - 15 Jun 2021

## 💥 Breaking Changes

- Add HashAlgorithm hash and hashWithTag functions ([#1002](https://github.com/onflow/cadence/issues/1002)) [@turbolent](https://github.com/turbolent)
- Update the Crypto contract ([#999](https://github.com/onflow/cadence/issues/999)) [@turbolent](https://github.com/turbolent)

## ⭐ Features

- Add HashAlgorithm hash and hashWithTag functions ([#1002](https://github.com/onflow/cadence/issues/1002)) [@turbolent](https://github.com/turbolent)
- Add npm package for the docgen-tool ([#1008](https://github.com/onflow/cadence/issues/1008)) [@SupunS](https://github.com/SupunS)
- Add wasm generation for docgen tool ([#1005](https://github.com/onflow/cadence/issues/1005)) [@SupunS](https://github.com/SupunS)

## 🛠 Improvements

- Make `PublicKey` type importable ([#995](https://github.com/onflow/cadence/issues/995)) [@SupunS](https://github.com/SupunS)
- Update the Crypto contract ([#999](https://github.com/onflow/cadence/issues/999)) [@turbolent](https://github.com/turbolent)
- Add dynamic type importability check for arguments ([#1007](https://github.com/onflow/cadence/issues/1007)) [@SupunS](https://github.com/SupunS)
- Language Server NPM package: Add support for Node environment, add tests ([#1006](https://github.com/onflow/cadence/issues/1006)) [@turbolent](https://github.com/turbolent)
- Embed markdown templates to docgen tool at compile time ([#1004](https://github.com/onflow/cadence/issues/1004)) [@SupunS](https://github.com/SupunS)
- No need for checking resource loss when function definitely halted ([#1000](https://github.com/onflow/cadence/issues/1000)) [@turbolent](https://github.com/turbolent)
- Remove the result declaration kind ([#1001](https://github.com/onflow/cadence/issues/1001)) [@turbolent](https://github.com/turbolent)
- Update for changes in Flowkit API ([#962](https://github.com/onflow/cadence/issues/962)) [@sideninja](https://github.com/sideninja)
- Add Supun as code owner ([#997](https://github.com/onflow/cadence/issues/997)) [@turbolent](https://github.com/turbolent)
- Disable the wasmtime VM for now ([#991](https://github.com/onflow/cadence/issues/991)) [@turbolent](https://github.com/turbolent)

## 🐞 Bug Fixes

- Add dynamic type importability check for arguments ([#1007](https://github.com/onflow/cadence/issues/1007)) [@SupunS](https://github.com/SupunS)
- Fix dictionary deferred owner ([#992](https://github.com/onflow/cadence/issues/992)) [@turbolent](https://github.com/turbolent)
- Fix AST walk for transaction declaration ([#998](https://github.com/onflow/cadence/issues/998)) [@turbolent](https://github.com/turbolent)

## 📖 Documentation

- Add documentation for the Cadence documentation generator ([#1003](https://github.com/onflow/cadence/issues/1003)) [@SupunS](https://github.com/SupunS)


[Changes][v0.18.0]


<a name="v0.17.0"></a>
# [v0.17.0](https://github.com/onflow/cadence/releases/tag/v0.17.0) - 09 Jun 2021

## ⭐ Features

- Add deferred decoding support for dictionary values ([#925](https://github.com/onflow/cadence/issues/925)) [@SupunS](https://github.com/SupunS)
- Track origin's occurrences ([#907](https://github.com/onflow/cadence/issues/907)) [@turbolent](https://github.com/turbolent)
- Declare more ranges ([#882](https://github.com/onflow/cadence/issues/882)) [@turbolent](https://github.com/turbolent)
- Language Server: Suggest completion items for declaration ranges ([#881](https://github.com/onflow/cadence/issues/881)) [@turbolent](https://github.com/turbolent)
- Provide code action to declare field and function ([#961](https://github.com/onflow/cadence/issues/961)) [@turbolent](https://github.com/turbolent)
- Provide code action to declare variable/constant ([#946](https://github.com/onflow/cadence/issues/946)) [@turbolent](https://github.com/turbolent)
- Add code action to implement missing members ([#942](https://github.com/onflow/cadence/issues/942)) [@turbolent](https://github.com/turbolent)
- Provide code actions / quick fixes ([#941](https://github.com/onflow/cadence/issues/941)) [@turbolent](https://github.com/turbolent)
- Provide documentation for range completion items ([#923](https://github.com/onflow/cadence/issues/923)) [@turbolent](https://github.com/turbolent)
- Record docstrings in variables  ([#922](https://github.com/onflow/cadence/issues/922)) [@turbolent](https://github.com/turbolent)
- Add a `String.utf8` field ([#954](https://github.com/onflow/cadence/issues/954)) [@turbolent](https://github.com/turbolent)
- Provide signature help ([#911](https://github.com/onflow/cadence/issues/911)) [@turbolent](https://github.com/turbolent)
- Add a String constructor function and a String.encodeHex function ([#953](https://github.com/onflow/cadence/issues/953)) [@turbolent](https://github.com/turbolent)
- Record position information for function invocations ([#910](https://github.com/onflow/cadence/issues/910)) [@turbolent](https://github.com/turbolent)
- Add function docs formatting support ([#938](https://github.com/onflow/cadence/issues/938)) [@SupunS](https://github.com/SupunS)
- Add Cadence documentation generator ([#927](https://github.com/onflow/cadence/issues/927)) [@SupunS](https://github.com/SupunS)
- Rename all occurrences of a symbol in the document ([#909](https://github.com/onflow/cadence/issues/909)) [@turbolent](https://github.com/turbolent)
- Enable DocumentSymbol capability for Outline ([#662](https://github.com/onflow/cadence/issues/662)) [@MaxStalker](https://github.com/MaxStalker)
- Add an AST walking function ([#939](https://github.com/onflow/cadence/issues/939)) [@turbolent](https://github.com/turbolent)
- Add HexToAddress utility function, add tests for address functions ([#932](https://github.com/onflow/cadence/issues/932)) [@turbolent](https://github.com/turbolent)
- Highlight all occurrences of a symbol in the document ([#908](https://github.com/onflow/cadence/issues/908)) [@turbolent](https://github.com/turbolent)

## 🛠 Improvements

- Add deferred decoding support for dictionary values ([#925](https://github.com/onflow/cadence/issues/925)) [@SupunS](https://github.com/SupunS)
- Improve type inference for binary expressions ([#957](https://github.com/onflow/cadence/issues/957)) [@turbolent](https://github.com/turbolent)
- Update language server to Cadence v0.16.0 ([#926](https://github.com/onflow/cadence/issues/926)) [@turbolent](https://github.com/turbolent)
- [Doc-Gen Tool] Add documentation generation support for event declarations ([#985](https://github.com/onflow/cadence/issues/985)) [@SupunS](https://github.com/SupunS)
- [Doc-Gen Tool] Group declarations based on their kind ([#984](https://github.com/onflow/cadence/issues/984)) [@SupunS](https://github.com/SupunS)
- Check 'importability' instead of 'storability' for transaction arguments ([#983](https://github.com/onflow/cadence/issues/983)) [@SupunS](https://github.com/SupunS)
- Allow native composite types to be passed as script arguments ([#973](https://github.com/onflow/cadence/issues/973)) [@SupunS](https://github.com/SupunS)
- Validate UTF-8 compatibility in string value constructor ([#972](https://github.com/onflow/cadence/issues/972)) [@SupunS](https://github.com/SupunS)
- Cache type ID for composite types ([#950](https://github.com/onflow/cadence/issues/950)) [@SupunS](https://github.com/SupunS)
- [Optimization] Make dynamic types singleton for simple types ([#963](https://github.com/onflow/cadence/issues/963)) [@SupunS](https://github.com/SupunS)
- Include docstrings in value and type variables, improve hover markup ([#945](https://github.com/onflow/cadence/issues/945)) [@turbolent](https://github.com/turbolent)
- Always check arguments and record the argument types in the elaboration ([#951](https://github.com/onflow/cadence/issues/951)) [@turbolent](https://github.com/turbolent)
- Optimize qualified identifier creation ([#948](https://github.com/onflow/cadence/issues/948)) [@SupunS](https://github.com/SupunS)
- Optimize `Type` function declaration ([#949](https://github.com/onflow/cadence/issues/949)) [@turbolent](https://github.com/turbolent)
- [Optimization] Re-use converter functions across interpreters ([#947](https://github.com/onflow/cadence/issues/947)) [@SupunS](https://github.com/SupunS)
- Update and extend the source compatibility suite ([#977](https://github.com/onflow/cadence/issues/977)) [@turbolent](https://github.com/turbolent)
- Update to Go 1.16.3's libexec/misc/wasm/wasm_exec.js ([#929](https://github.com/onflow/cadence/issues/929)) [@turbolent](https://github.com/turbolent)

## 🐞 Bug Fixes

- Track origin's occurrences ([#907](https://github.com/onflow/cadence/issues/907)) [@turbolent](https://github.com/turbolent)
- Bring back fmt.Stringer implementation for interpreter.Value ([#969](https://github.com/onflow/cadence/issues/969)) [@turbolent](https://github.com/turbolent)
- Fix the initialization order in the interpreter ([#958](https://github.com/onflow/cadence/issues/958)) [@turbolent](https://github.com/turbolent)
- Fix parsing of negative fractional fixed-point strings ([#935](https://github.com/onflow/cadence/issues/935)) [@mikeylemmon](https://github.com/mikeylemmon)
- capture computation used for all transactions ([#895](https://github.com/onflow/cadence/issues/895)) [@ramtinms](https://github.com/ramtinms)
- Only wrap a type in an optional if it is not nil ([#937](https://github.com/onflow/cadence/issues/937)) [@turbolent](https://github.com/turbolent)

## 🧪 Tests

- Add test case for invalid utf8 string import ([#968](https://github.com/onflow/cadence/issues/968)) [@SupunS](https://github.com/SupunS)
- Test contract member initialization ([#931](https://github.com/onflow/cadence/issues/931)) [@turbolent](https://github.com/turbolent)

## 📖 Documentation

- Update account docs to not to pass PublicKey as an argument ([#967](https://github.com/onflow/cadence/issues/967)) [@SupunS](https://github.com/SupunS)
- Improve the crypto doc ([#960](https://github.com/onflow/cadence/issues/960)) [@tarakby](https://github.com/tarakby)

## 🏗 Chore

- Use actions/setup-go@v2 to speed up CI by ~10 secs and specify Go 1.15.x ([#971](https://github.com/onflow/cadence/issues/971)) [@fxamacker](https://github.com/fxamacker)
- Remove italics from auto flow-go PR ([#988](https://github.com/onflow/cadence/issues/988)) [@janezpodhostnik](https://github.com/janezpodhostnik)
- Improve sync flow_go action ([#987](https://github.com/onflow/cadence/issues/987)) [@janezpodhostnik](https://github.com/janezpodhostnik)
- Fix for auto-update flow-go: no fail on tidy ([#986](https://github.com/onflow/cadence/issues/986)) [@janezpodhostnik](https://github.com/janezpodhostnik)
- Auto update flow-go github action ([#981](https://github.com/onflow/cadence/issues/981)) [@janezpodhostnik](https://github.com/janezpodhostnik)
- Remove coverage of empty functions from report ([#944](https://github.com/onflow/cadence/issues/944)) [@turbolent](https://github.com/turbolent)
- Improve coverage calculation ([#933](https://github.com/onflow/cadence/issues/933)) [@turbolent](https://github.com/turbolent)

[Changes][v0.17.0]


<a name="v0.16.1"></a>
# [v0.16.1](https://github.com/onflow/cadence/releases/tag/v0.16.1) - 23 May 2021

## 🐞 Bug Fixes

- Only wrap a type in an optional if it is not nil ([#936](https://github.com/onflow/cadence/issues/936)) [@SupunS](https://github.com/SupunS) 


[Changes][v0.16.1]


<a name="v0.16.0"></a>
# [v0.16.0](https://github.com/onflow/cadence/releases/tag/v0.16.0) - 18 May 2021


## 💥 Breaking Changes

- Add new crypto features ([#852](https://github.com/onflow/cadence/issues/852)) [@SupunS](https://github.com/SupunS)

  Renamed the `ECDSA_Secp256k1` signature algorithm to `ECDSA_secp256k1`.

  Please update existing contracts, transactions, and scripts to use the new name.

- Remove the high-level storage API ([#877](https://github.com/onflow/cadence/issues/877)) [@turbolent](https://github.com/turbolent)

  The internal interface's `SetCadenceValue` function was removed.

  This change does not affect Cadence programs (contracts, transactions, scripts).

## ⭐ Features

- Add non-local type inference for expressions ([#875](https://github.com/onflow/cadence/issues/875)) [@SupunS](https://github.com/SupunS)

  The type of most declarations and expressions is now inferred in a uni-directional way, instead of only locally.

  For example, this allows the following declarations to type-check without having to add static casts:

  ```kotlin
  let numbers: [Int8] = [1, 2, 3]
  let nestedNumbers: [[Int8]] = [[1, 2], [3, 4]]
  let numberNames: {Int64: String} = {0: "zero", 1: "one"}
  let sum: Int8 = 1 + 2
  ```

- Add new crypto features ([#852](https://github.com/onflow/cadence/issues/852)) [@SupunS](https://github.com/SupunS)

  **Hash Algorithms**

  - Added [KMAC128_BLS_BLS12_381 hashing algorithm](https://docs.onflow.org/cadence/language/crypto/#hashing-algorithms)): `HashAlgorithm.KMAC128_BLS_BLS12_381`

  **Signature Algorithms**

  - Added [BLS_BLS12_381 signature algorithm](https://docs.onflow.org/cadence/language/crypto/#signature-algorithms): `SignatureAlgorithm.BLS_BLS12_381`

  **`PublicKey` Type**

  - Added the field `let isValid: Bool`. For example:

    ```kotlin
    let publicKey = PublicKey(publicKey: [], signatureAlgorithm: SignatureAlgorithm.ECDSA_P256)

    let valid = publicKey.isValid
    ```

  - Added the function `verify`, which allows validating signatures:

    ```kotlin
    pub fun verify(
        signature: [UInt8],
        signedData: [UInt8],
        domainSeparationTag: String,
        hashAlgorithm: HashAlgorithm
    ): Bool
    ```

    For example:

    ```kotlin
    let valid = publicKey.verify(
        signature: [],
        signedData: [],
        domainSeparationTag: "something",
        hashAlgorithm: HashAlgorithm.SHA2_256
    )
    ```

  - Added the function `hashWithTag`, which allows hashing with a tag:

    ```kotlin
    pub fun hashWithTag(
        _ data: [UInt8],
        tag: string,
        algorithm: HashAlgorithm
    ): [UInt8]
    ```

- Direct contract function invoke ([#878](https://github.com/onflow/cadence/issues/878)) [@janezpodhostnik](https://github.com/janezpodhostnik)

  Cadence now supports an additional function to allow the host environment to call contract functions directly.

- Language Server: Add support for access check mode ([#868](https://github.com/onflow/cadence/issues/868)) [@turbolent](https://github.com/turbolent)
- Record variable declaration ranges ([#880](https://github.com/onflow/cadence/issues/880)) [@turbolent](https://github.com/turbolent)

## ⚡️ Performance

The performance of decoding and encoding values was significantly improved by [@fxamacker](https://github.com/fxamacker) and [@SupunS](https://github.com/SupunS):

- Optimize encoding by using CBOR lib's StreamEncoder ([#830](https://github.com/onflow/cadence/issues/830)) [@fxamacker](https://github.com/fxamacker)
- Optimize decoding by using CBOR StreamDecoder ([#885](https://github.com/onflow/cadence/issues/885)) [@fxamacker](https://github.com/fxamacker)
- Add deferred decoding for array values ([#871](https://github.com/onflow/cadence/issues/871)) [@SupunS](https://github.com/SupunS)
- Add deferred decoding support for composite values ([#896](https://github.com/onflow/cadence/issues/896)) [@SupunS](https://github.com/SupunS)
- Pre-allocate and reuse value-path for encoding ([#858](https://github.com/onflow/cadence/issues/858)) [@fxamacker](https://github.com/fxamacker)
- Pre-allocate and reuse value-path for decoding ([#869](https://github.com/onflow/cadence/issues/869)) [@turbolent](https://github.com/turbolent)
- Optimize Address.Bytes() to inline fast path ([#848](https://github.com/onflow/cadence/issues/848)) [@fxamacker](https://github.com/fxamacker)
- Remove call to Valid() during encoding to boost speed by about 13-18% ([#857](https://github.com/onflow/cadence/issues/857)) [@fxamacker](https://github.com/fxamacker)
- Directly create int subtype value at interpreter ([#913](https://github.com/onflow/cadence/issues/913)) [@SupunS](https://github.com/SupunS)

## 🛠 Improvements

- Refactor Language Server to use CLI shared library ([#751](https://github.com/onflow/cadence/issues/751)) [@MaxStalker](https://github.com/MaxStalker)
- Make PublicKey value immutable ([#879](https://github.com/onflow/cadence/issues/879)) [@SupunS](https://github.com/SupunS)
- Language Server: update flow-cli ([#894](https://github.com/onflow/cadence/issues/894)) [@psiemens](https://github.com/psiemens)
- Update language server to latest Cadence and Go SDK ([#874](https://github.com/onflow/cadence/issues/874)) [@psiemens](https://github.com/psiemens)
- Make the interpreter location optional ([#918](https://github.com/onflow/cadence/issues/918)) [@turbolent](https://github.com/turbolent)

## 🔒 Security

This release contains major security fixes:

- Declare post-condition result variable as reference if return type is a resource ([#905](https://github.com/onflow/cadence/issues/905)) [@turbolent](https://github.com/turbolent)

  We would like to thank Deniz Mert Edincik for finding and reporting this critical issue responsibly through our [Responsible Disclosure Policy](https://github.com/onflow/cadence/blob/master/SECURITY.md).

  See https://forum.onflow.org/t/fixed-cadence-vulnerability-2021-04-13

- Fix dynamic subtype test for reference values ([#914](https://github.com/onflow/cadence/issues/914)) [@turbolent](https://github.com/turbolent)

  We would like to thank Deniz Mert Edincik for finding and reporting this critical issue responsibly through our [Responsible Disclosure Policy](https://github.com/onflow/cadence/blob/master/SECURITY.md).

  We would also like to thank Mikey Lemmon for finding this issue independently and reporting it responsibly, too.

  See https://forum.onflow.org/t/fixed-cadence-vulnerability-2021-04-20

- Check storability when value is written (before it is encoded) ([#915](https://github.com/onflow/cadence/issues/915)) [@turbolent](https://github.com/turbolent)

  We would like to thank Deniz Mert Edincik for finding and reporting this medium issue responsibly through our [Responsible Disclosure Policy](https://github.com/onflow/cadence/blob/master/SECURITY.md).

  See https://forum.onflow.org/t/fixed-cadence-vulnerability-2021-04-20

- Check the index bounds for arrays ([#917](https://github.com/onflow/cadence/issues/917)) [@turbolent](https://github.com/turbolent)

  We would like to thank Mário Silva for finding and reporting this medium issue responsibly through our [Responsible Disclosure Policy](https://github.com/onflow/cadence/blob/master/SECURITY.md).

## 🐞 Bug Fixes

- Improve reference values' dynamic type, static type, copy, equal, and conformance functions ([#921](https://github.com/onflow/cadence/issues/921)) [@turbolent](https://github.com/turbolent)
- Implement `StaticType` for `StorageReferenceValue` and `EphemeralReferenceValue` ([#920](https://github.com/onflow/cadence/issues/920)) [@turbolent](https://github.com/turbolent)
- Fix strings ([#919](https://github.com/onflow/cadence/issues/919)) [@turbolent](https://github.com/turbolent)
- PublicKeyValue stringer fix ([#903](https://github.com/onflow/cadence/issues/903)) [@janezpodhostnik](https://github.com/janezpodhostnik)
- Handle recursive values (due to references) in string function ([#906](https://github.com/onflow/cadence/issues/906)) [@turbolent](https://github.com/turbolent)
- Handle case with no arguments ([#886](https://github.com/onflow/cadence/issues/886)) [@MaxStalker](https://github.com/MaxStalker)
- Fix link function return value ([#865](https://github.com/onflow/cadence/issues/865)) [@turbolent](https://github.com/turbolent)
- Fix optional type ID, add tests ([#864](https://github.com/onflow/cadence/issues/864)) [@turbolent](https://github.com/turbolent)
- Fix function type members  ([#867](https://github.com/onflow/cadence/issues/867)) [@turbolent](https://github.com/turbolent)
- Language Server:  Reuse existing checkers instead of re-parsing and re-checking imports  ([#855](https://github.com/onflow/cadence/issues/855)) [@turbolent](https://github.com/turbolent)
- Only consider the program invalid if there are error diagnostics ([#854](https://github.com/onflow/cadence/issues/854)) [@turbolent](https://github.com/turbolent)

## 🧪 Tests

- Add test for TopShot moment batch transfer ([#826](https://github.com/onflow/cadence/issues/826)) [@turbolent](https://github.com/turbolent)
- Test equal for nil value, in comparison and switch ([#866](https://github.com/onflow/cadence/issues/866)) [@turbolent](https://github.com/turbolent)
- Test storing capabilities ([#856](https://github.com/onflow/cadence/issues/856)) [@turbolent](https://github.com/turbolent)
- Test getCapability after unlink ([#849](https://github.com/onflow/cadence/issues/849)) [@turbolent](https://github.com/turbolent)
- Add interpreter tests for common resource reference usages ([#916](https://github.com/onflow/cadence/issues/916)) [@turbolent](https://github.com/turbolent)
- Add extra test for deterministic JSON export ([#893](https://github.com/onflow/cadence/issues/893)) [@m4ksio](https://github.com/m4ksio)

## 📖 Documentation

- Add a reference to PublicKey in the crypto docs ([#912](https://github.com/onflow/cadence/issues/912)) [@SupunS](https://github.com/SupunS)
- Tutorial typo fixes ([#825](https://github.com/onflow/cadence/issues/825)) [@joshuahannan](https://github.com/joshuahannan)
- Update values-and-types.mdx ([#827](https://github.com/onflow/cadence/issues/827)) [@FeiyangTan](https://github.com/FeiyangTan)

## 🏗 Chores

- Enable more linters and lint ([#902](https://github.com/onflow/cadence/issues/902)) [@turbolent](https://github.com/turbolent)
- Measure and report code coverage ([#899](https://github.com/onflow/cadence/issues/899)) [@turbolent](https://github.com/turbolent)
- Fix CI linter failure ([#904](https://github.com/onflow/cadence/issues/904)) [@SupunS](https://github.com/SupunS)
- Fix Codecov "file not found at ./coverage.txt" ([#901](https://github.com/onflow/cadence/issues/901)) [@fxamacker](https://github.com/fxamacker)


[Changes][v0.16.0]


<a name="v0.15.1"></a>
# [v0.15.1](https://github.com/onflow/cadence/releases/tag/v0.15.1) - 21 Apr 2021

## 🛠 Improvements

- Include wrapped error in InvalidEntryPointArgumentError message ([#823](https://github.com/onflow/cadence/issues/823)) [@mikeylemmon](https://github.com/mikeylemmon)
- Remove encoding and decoding support for storage references ([#819](https://github.com/onflow/cadence/issues/819)) [@turbolent](https://github.com/turbolent)

## 📖 Documentation

- Document saturation arithmetic and min/max fields ([#817](https://github.com/onflow/cadence/issues/817)) [@turbolent](https://github.com/turbolent)


[Changes][v0.15.1]


<a name="v0.15.0"></a>
# [v0.15.0](https://github.com/onflow/cadence/releases/tag/v0.15.0) - 20 Apr 2021

## ⭐ Features

- Added balance fields on accounts ([#808](https://github.com/onflow/cadence/issues/808)) [@janezpodhostnik](https://github.com/janezpodhostnik)
- Add address field to capabilities ([#736](https://github.com/onflow/cadence/issues/736)) [@ceelo777](https://github.com/ceelo777)
- Add array appendAll function ([#727](https://github.com/onflow/cadence/issues/727)) [@lkadian](https://github.com/lkadian)
- Validate arguments passed into cadence programs ([#724](https://github.com/onflow/cadence/issues/724)) [@SupunS](https://github.com/SupunS)
- Implement equality for storable values and static types ([#790](https://github.com/onflow/cadence/issues/790)) [@turbolent](https://github.com/turbolent)
- Declare min/max fields for integer and fixed-point types ([#803](https://github.com/onflow/cadence/issues/803)) [@turbolent](https://github.com/turbolent)
- Add saturation arithmetic ([#804](https://github.com/onflow/cadence/issues/804)) [@turbolent](https://github.com/turbolent)
- Add functions to read stored and linked values ([#812](https://github.com/onflow/cadence/issues/812)) [@turbolent](https://github.com/turbolent)

## 🛠 Improvements

- Optimize storage format ([#797](https://github.com/onflow/cadence/issues/797)) [@fxamacker](https://github.com/fxamacker)
- Paralleize encoding ([#731](https://github.com/onflow/cadence/issues/731)) [@zhangchiqing](https://github.com/zhangchiqing)
- Deny removing contracts if contract update validation is enabled ([#792](https://github.com/onflow/cadence/issues/792)) [@SupunS](https://github.com/SupunS)
- Simplify function types ([#802](https://github.com/onflow/cadence/issues/802)) [@turbolent](https://github.com/turbolent)
- Cache capability type members ([#799](https://github.com/onflow/cadence/issues/799)) [@turbolent](https://github.com/turbolent)
- Use force expression's end position in force nil error ([#789](https://github.com/onflow/cadence/issues/789)) [@turbolent](https://github.com/turbolent)
- Handle cyclic references in dynamic type conformance check ([#787](https://github.com/onflow/cadence/issues/787)) [@SupunS](https://github.com/SupunS)
- Benchmark CBOR encoding/decoding using mainnet data ([#783](https://github.com/onflow/cadence/issues/783)) [@fxamacker](https://github.com/fxamacker)
- Extend defensive run-time checking to all value transfers ([#784](https://github.com/onflow/cadence/issues/784)) [@turbolent](https://github.com/turbolent)
- Remove obsolete code in decode.go ([#788](https://github.com/onflow/cadence/issues/788)) [@fxamacker](https://github.com/fxamacker)
- Optimize encoding of 12 types still using CBOR map ([#778](https://github.com/onflow/cadence/issues/778)) [@fxamacker](https://github.com/fxamacker)
- Reject indirect incompatible updates to composite declarations ([#772](https://github.com/onflow/cadence/issues/772)) [@SupunS](https://github.com/SupunS)
- Panic with a dedicated error when a member access results in no value ([#768](https://github.com/onflow/cadence/issues/768)) [@turbolent](https://github.com/turbolent)
- Update language server to Cadence v0.14.4 ([#767](https://github.com/onflow/cadence/issues/767)) [@turbolent](https://github.com/turbolent)
- Improve value type conformance check ([#776](https://github.com/onflow/cadence/issues/776)) [@turbolent](https://github.com/turbolent)
- Add validation for enum cases during contract updates ([#762](https://github.com/onflow/cadence/issues/762)) [@SupunS](https://github.com/SupunS)
- Optimize encoding of composites ([#761](https://github.com/onflow/cadence/issues/761)) [@fxamacker](https://github.com/fxamacker)
- Improve state decoding tool ([#759](https://github.com/onflow/cadence/issues/759)) [@turbolent](https://github.com/turbolent)
- Optimize encoding of dictionaries ([#752](https://github.com/onflow/cadence/issues/752)) [@fxamacker](https://github.com/fxamacker)
- Prepare Decoder for storage format v4 ([#746](https://github.com/onflow/cadence/issues/746)) [@fxamacker](https://github.com/fxamacker)
- Make numeric types singleton ([#732](https://github.com/onflow/cadence/issues/732)) [@SupunS](https://github.com/SupunS)
- Validate arguments passed into cadence programs ([#724](https://github.com/onflow/cadence/issues/724)) [@SupunS](https://github.com/SupunS)
- Use location ID as key for maps holding account codes ([#723](https://github.com/onflow/cadence/issues/723)) [@turbolent](https://github.com/turbolent)
- Make native composite types non-storable ([#713](https://github.com/onflow/cadence/issues/713)) [@turbolent](https://github.com/turbolent)
- Improve state decoding tool ([#798](https://github.com/onflow/cadence/issues/798)) [@turbolent](https://github.com/turbolent)
- Remove unused storage key handler ([#811](https://github.com/onflow/cadence/issues/811)) [@turbolent](https://github.com/turbolent)

## 🐞 Bug Fixes

- Extend defensive run-time checking to all value transfers ([#784](https://github.com/onflow/cadence/issues/784)) [@turbolent](https://github.com/turbolent)
- Fix borrowing ([#782](https://github.com/onflow/cadence/issues/782)) [@turbolent](https://github.com/turbolent)
- Prevent cyclic imports ([#809](https://github.com/onflow/cadence/issues/809)) [@turbolent](https://github.com/turbolent)
- Clean up "storage removal" index expression ([#769](https://github.com/onflow/cadence/issues/769)) [@turbolent](https://github.com/turbolent)
- Get resource owner from host environment ([#770](https://github.com/onflow/cadence/issues/770)) [@SupunS](https://github.com/SupunS)
- Handle field initialization using force-move assignment ([#741](https://github.com/onflow/cadence/issues/741)) [@turbolent](https://github.com/turbolent)
- Fix error range when reporting a type error for an indexing reference expression ([#719](https://github.com/onflow/cadence/issues/719)) [@turbolent](https://github.com/turbolent)

## 📖 Documentation

- Improve documentation for block timestamp ([#775](https://github.com/onflow/cadence/issues/775)) [@turbolent](https://github.com/turbolent)
- Document run-time type identifiers ([#750](https://github.com/onflow/cadence/issues/750)) [@turbolent](https://github.com/turbolent)
- Add documentation for the getType() builtin method  ([#737](https://github.com/onflow/cadence/issues/737)) [@SupunS](https://github.com/SupunS)


[Changes][v0.15.0]


<a name="v0.14.5"></a>
# [v0.14.5](https://github.com/onflow/cadence/releases/tag/v0.14.5) - 09 Apr 2021

## 🐞 Bug Fixes

- Fix borrowing ([#785](https://github.com/onflow/cadence/issues/785))

[Changes][v0.14.5]


<a name="v0.14.4"></a>
# [v0.14.4](https://github.com/onflow/cadence/releases/tag/v0.14.4) - 25 Mar 2021

## ⭐ Features

- Add dictionary contains function ([#716](https://github.com/onflow/cadence/issues/716)) [@lkadian](https://github.com/lkadian)

## 🐞 Bug Fixes

- Fix runtime representation of native HashAlgorithm/SignAlgorithm enums ([#725](https://github.com/onflow/cadence/issues/725)) [@SupunS](https://github.com/SupunS)
- Fix error range when reporting a type error for an indexing reference expression ([#719](https://github.com/onflow/cadence/issues/719)) [@turbolent](https://github.com/turbolent)

## 🧪 Tests

- Benchmark real fungible token transfers ([#722](https://github.com/onflow/cadence/issues/722)) [@turbolent](https://github.com/turbolent)
- Use location ID as key for maps holding account codes ([#723](https://github.com/onflow/cadence/issues/723)) [@turbolent](https://github.com/turbolent)

[Changes][v0.14.4]


<a name="v0.13.10"></a>
# [v0.13.10](https://github.com/onflow/cadence/releases/tag/v0.13.10) - 25 Mar 2021

## 🐞 Bug Fixes

- Add support for access(account) and multiple contracts per account ([#730](https://github.com/onflow/cadence/issues/730)) [@turbolent](https://github.com/turbolent)

[Changes][v0.13.10]


<a name="v0.13.9"></a>
# [v0.13.9](https://github.com/onflow/cadence/releases/tag/v0.13.9) - 23 Mar 2021

## 🛠 Improvements

- Lazily load contract values ([#720](https://github.com/onflow/cadence/issues/720)) [@turbolent](https://github.com/turbolent)


[Changes][v0.13.9]


<a name="v0.14.3"></a>
# [v0.14.3](https://github.com/onflow/cadence/releases/tag/v0.14.3) - 22 Mar 2021

## 🛠 Improvements

- Lazily load contract values ([#715](https://github.com/onflow/cadence/issues/715)) [@turbolent](https://github.com/turbolent)
- Make native composite types non-storable ([#713](https://github.com/onflow/cadence/issues/713)) [@turbolent](https://github.com/turbolent)
- Ensure imported checkers/elaborations are reused in tests' import handlers ([#711](https://github.com/onflow/cadence/issues/711)) [@turbolent](https://github.com/turbolent)
- Use require.ErrorAs ([#714](https://github.com/onflow/cadence/issues/714)) [@turbolent](https://github.com/turbolent)

## 🐞 Bug Fixes

- Add support for access(account) and multiple contracts per account ([#710](https://github.com/onflow/cadence/issues/710)) [@turbolent](https://github.com/turbolent)


[Changes][v0.14.3]


<a name="v0.14.2"></a>
# [v0.14.2](https://github.com/onflow/cadence/releases/tag/v0.14.2) - 18 Mar 2021

## ⭐ Features

- Replace parser demo with AST explorer ([#693](https://github.com/onflow/cadence/issues/693)) [@turbolent](https://github.com/turbolent)
- Report a hint when a dynamic cast is statically known to always succeed ([#688](https://github.com/onflow/cadence/issues/688)) [@turbolent](https://github.com/turbolent)

## 🛠 Improvements

- Revert the addition of the unsupported signing and hash algorithms ([#705](https://github.com/onflow/cadence/issues/705)) [@turbolent](https://github.com/turbolent)

## 🐞 Bug Fixes

- Fix cadence-parser NPM package, update versions ([#694](https://github.com/onflow/cadence/issues/694)) [@turbolent](https://github.com/turbolent)
- Properly handle recursive types when checking validity of event parameter types ([#709](https://github.com/onflow/cadence/issues/709)) [@turbolent](https://github.com/turbolent)
- Make all path sub-types externally returnable ([#707](https://github.com/onflow/cadence/issues/707)) [@turbolent](https://github.com/turbolent)

## 📖 Documentation

- Fix the syntax of the signature algorithm enum in docs ([#701](https://github.com/onflow/cadence/issues/701)) [@SupunS](https://github.com/SupunS)


[Changes][v0.14.2]


<a name="v0.13.8"></a>
# [v0.13.8](https://github.com/onflow/cadence/releases/tag/v0.13.8) - 18 Mar 2021

## 🐞 Bug Fixes

- Properly handle recursive types when checking validity of event parameter types ([#708](https://github.com/onflow/cadence/issues/708)) [@turbolent](https://github.com/turbolent)

[Changes][v0.13.8]


<a name="v0.14.1"></a>
# [v0.14.1](https://github.com/onflow/cadence/releases/tag/v0.14.1) - 16 Mar 2021

## 🛠 Improvements

- Add unknown constants for signature algorithm and hash algorithm ([#699](https://github.com/onflow/cadence/issues/699)) [@turbolent](https://github.com/turbolent)
- Optimize checker activations ([#674](https://github.com/onflow/cadence/issues/674)) [@turbolent](https://github.com/turbolent)
- Return nil when revoking a non-existing key ([#697](https://github.com/onflow/cadence/issues/697)) [@SupunS](https://github.com/SupunS)

## 🐞 Bug Fixes

- Fix nested error pretty printing ([#695](https://github.com/onflow/cadence/issues/695)) [@turbolent](https://github.com/turbolent)


[Changes][v0.14.1]


<a name="v0.14.0"></a>
# [v0.14.0](https://github.com/onflow/cadence/releases/tag/v0.14.0) - 15 Mar 2021

This release introduced a new [high-level Account Key API](https://docs.onflow.org/cadence/language/accounts/) and improves the interpreter performance.
The current low-level Account Key API is now deprecated and will be removed in a future release. Please switch to the new one.

The following example transaction demonstrates the functionality:

```kotlin
transaction {
	prepare(signer: AuthAccount) {
		let newPublicKey = PublicKey(
			publicKey: "010203".decodeHex(),
			signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
		)

		// Add a key
		let addedKey = signer.keys.add(
			publicKey: newPublicKey,
			hashAlgorithm: HashAlgorithm.SHA3_256,
			weight: 100.0
		)


		// Retrieve a key
		let sameKey = signer.keys.get(keyIndex: addedKey.keyIndex)


		// Revoke a key
		signer.keys.revoke(keyIndex: addedKey.keyIndex)
	}
}
```

## ⭐ Features

- Introduce a new [high-level Account Key API](https://docs.onflow.org/cadence/language/accounts/) ([#633](https://github.com/onflow/cadence/issues/633)) [@SupunS](https://github.com/SupunS)
- Allow the Language Server to be debugged ([#663](https://github.com/onflow/cadence/issues/663)) [@turbolent](https://github.com/turbolent)

## 💥 Breaking Changes

The `Crypto` contract has changed in a backwards-incompatible way, so the types and values it declared could be used in the new Account Key API:

- The struct `Crypto.PublicKey` was replaced by the new built-in global struct `PublicKey`. 
  There is no need anymore to import the `Crypto` contract to work with public keys.

- The struct `Crypto.SignatureAlgorithm` was replaced with the new built-in global enum `SignatureAlgorithm`.
  There is no need anymore to import the `Crypto` contract to work with signature algorithms.

- The struct `Crypto.HashAlgorithm` was replaced with the new built-in global enum `HashAlgorithm`.
  There is no need anymore to import the `Crypto` contract to work with hash algorithms.

- The signature algorithm value `Crypto.ECDSA_Secp256k1` was replaced with the new built-in enum case `SignatureAlgorithm.ECDSA_Secp256k1`

- The signature algorithm value `Crypto.ECDSA_P256` was replaced with the new built-in enum case `SignatureAlgorithm.ECDSA_P256`

- The hash algorithm `Crypto.SHA3_256` was replaced with the new built-in enum case `HashAlgorithm.SHA3_256`

- The hash algorithm `Crypto.SHA2_256` was replaced with the new built-in enum case `HashAlgorithm.SHA2_256`

## 🛠 Improvements

- Add support for importing enum values ([#672](https://github.com/onflow/cadence/issues/672)) [@SupunS](https://github.com/SupunS)
- Optimize interpreter: Make invocation location ranges lazy ([#685](https://github.com/onflow/cadence/issues/685)) [@turbolent](https://github.com/turbolent)
- Optimize interpreter activations ([#673](https://github.com/onflow/cadence/issues/673)) [@turbolent](https://github.com/turbolent)
- Optimize integer conversion ([#677](https://github.com/onflow/cadence/issues/677)) [@turbolent](https://github.com/turbolent)
- Optimize interpreter: Evaluate statements and declarations directly, remove trampolines ([#684](https://github.com/onflow/cadence/issues/684)) [@turbolent](https://github.com/turbolent)
- Optimize interpreter: Move statement evaluation code  ([#683](https://github.com/onflow/cadence/issues/683)) [@turbolent](https://github.com/turbolent)
- Optimize interpreter: Move evaluation of expressions and statements to separate files ([#682](https://github.com/onflow/cadence/issues/682)) [@turbolent](https://github.com/turbolent)
- Optimize interpreter: Evaluate expressions directly without trampolines ([#681](https://github.com/onflow/cadence/issues/681)) [@turbolent](https://github.com/turbolent)
- Optimize interpreter: Refactor function invocations to return a value instead of a trampoline  ([#680](https://github.com/onflow/cadence/issues/680)) [@turbolent](https://github.com/turbolent)
- Optimize interpreter: Evaluate binary expressions directly ([#679](https://github.com/onflow/cadence/issues/679)) [@turbolent](https://github.com/turbolent)

## 🐞 Bug Fixes

- Add support for exporting enums ([#669](https://github.com/onflow/cadence/issues/669)) [@SupunS](https://github.com/SupunS)
- Ensure code is long enough when extracting excerpt ([#690](https://github.com/onflow/cadence/issues/690)) [@turbolent](https://github.com/turbolent)

## 📖 Documentation

- Add documentation for the new account key API ([#635](https://github.com/onflow/cadence/issues/635)) [@SupunS](https://github.com/SupunS)






[Changes][v0.14.0]


<a name="v0.13.6"></a>
# [v0.13.6](https://github.com/onflow/cadence/releases/tag/v0.13.6) - 09 Mar 2021

## 🐞 Bug Fixes

- Revert "Cache qualified identifier and type ID for composite types and interface types" ([#670](https://github.com/onflow/cadence/issues/670)) [@turbolent](https://github.com/turbolent)


[Changes][v0.13.6]


<a name="v0.13.5"></a>
# [v0.13.5](https://github.com/onflow/cadence/releases/tag/v0.13.5) - 05 Mar 2021

## 🛠 Improvements

- Add computed fields to the composite value ([#664](https://github.com/onflow/cadence/issues/664)) [@SupunS](https://github.com/SupunS)
- Improve naming: Rename nominal type to simple type ([#659](https://github.com/onflow/cadence/issues/659)) [@turbolent](https://github.com/turbolent)
- Cache qualified identifier and type ID for composite types and interface types ([#658](https://github.com/onflow/cadence/issues/658)) [@turbolent](https://github.com/turbolent)

## 🐞 Bug Fixes

- Fix handling of capability values without borrow type ([#666](https://github.com/onflow/cadence/issues/666)) [@turbolent](https://github.com/turbolent)
- Improve type equality check in the contract update validation ([#654](https://github.com/onflow/cadence/issues/654)) [@SupunS](https://github.com/SupunS)


[Changes][v0.13.5]


<a name="v0.13.4"></a>
# [v0.13.4](https://github.com/onflow/cadence/releases/tag/v0.13.4) - 04 Mar 2021

## ⭐ Features

- Add parser for pragma signers ([#656](https://github.com/onflow/cadence/issues/656)) [@MaxStalker](https://github.com/MaxStalker)

## 🐞 Bug Fixes

- Fix updating contracts with reference typed fields ([#649](https://github.com/onflow/cadence/issues/649)) [@SupunS](https://github.com/SupunS)

## 📖 Documentation

- Move remaining Cadence docs to Cadence repo. ([#653](https://github.com/onflow/cadence/issues/653)) [@10thfloor](https://github.com/10thfloor)


[Changes][v0.13.4]


<a name="v0.13.3"></a>
# [v0.13.3](https://github.com/onflow/cadence/releases/tag/v0.13.3) - 03 Mar 2021

## 🛠 Improvements

- Optimize checker construction ([#630](https://github.com/onflow/cadence/issues/630)) [@turbolent](https://github.com/turbolent)


[Changes][v0.13.3]


<a name="v0.13.2"></a>
# [v0.13.2](https://github.com/onflow/cadence/releases/tag/v0.13.2) - 03 Mar 2021

## 🛠 Improvements

- Make contract update validation optional and disable it by default ([#646](https://github.com/onflow/cadence/issues/646)) [@turbolent](https://github.com/turbolent)
- Delay contract code and value updates to the end of execution ([#636](https://github.com/onflow/cadence/issues/636)) [@turbolent](https://github.com/turbolent)
- Optimize encoding of positive bigints ([#637](https://github.com/onflow/cadence/issues/637)) [@turbolent](https://github.com/turbolent)
- Optimize deferred dictionary keys ([#638](https://github.com/onflow/cadence/issues/638)) [@turbolent](https://github.com/turbolent)
- Refactor `String`, `AuthAccount.Contracts`, and `DeployedContract` type to singleton ([#625](https://github.com/onflow/cadence/issues/625)) [@turbolent](https://github.com/turbolent)
- Refactor `AuthAccount`, `PublicAccount`, and `Block` type to singleton ([#624](https://github.com/onflow/cadence/issues/624)) [@turbolent](https://github.com/turbolent)
- Only record member accesses when origins and occurrences are enabled ([#627](https://github.com/onflow/cadence/issues/627)) [@turbolent](https://github.com/turbolent)
- Cache members for array and dictionary types ([#626](https://github.com/onflow/cadence/issues/626)) [@turbolent](https://github.com/turbolent)

## 🐞 Bug Fixes

- Support nested file imports in LSP ([#616](https://github.com/onflow/cadence/issues/616)) [@psiemens](https://github.com/psiemens)

## 📖 Documentation

- Fix code examples and improve documentation in language reference ([#634](https://github.com/onflow/cadence/issues/634)) [@jeroenlm](https://github.com/jeroenlm)


[Changes][v0.13.2]


<a name="v0.13.1"></a>
# [v0.13.1](https://github.com/onflow/cadence/releases/tag/v0.13.1) - 25 Feb 2021

## 🛠 Improvements

- Remove prepare and decode callbacks ([#622](https://github.com/onflow/cadence/issues/622)) [@turbolent](https://github.com/turbolent)
- Update language server to Cadence v0.13.0 and Go SDK v0.15.0 ([#623](https://github.com/onflow/cadence/issues/623)) [@turbolent](https://github.com/turbolent)
- Refactor Any type, AnyResource type, and AnyStruct type to singleton ([#618](https://github.com/onflow/cadence/issues/618)) [@turbolent](https://github.com/turbolent)
- Refactor Type, Bool, and Character type to singletons ([#617](https://github.com/onflow/cadence/issues/617)) [@turbolent](https://github.com/turbolent)

## 🐞 Bug Fixes

- Fix AuthAccount nested types ([#619](https://github.com/onflow/cadence/issues/619)) [@turbolent](https://github.com/turbolent)


[Changes][v0.13.1]


<a name="v0.12.12"></a>
# [v0.12.12](https://github.com/onflow/cadence/releases/tag/v0.12.12) - 24 Feb 2021

## 🛠 Improvements

- Remove the prepare callback ([#621](https://github.com/onflow/cadence/issues/621)) [@turbolent](https://github.com/turbolent)

[Changes][v0.12.12]


<a name="v0.13.0"></a>
# [v0.13.0](https://github.com/onflow/cadence/releases/tag/v0.13.0) - 22 Feb 2021

## ⭐ Features

- Validate contract updates ([#593](https://github.com/onflow/cadence/issues/593)) [@SupunS](https://github.com/SupunS)
- WebAssembly: Use reference types ([#448](https://github.com/onflow/cadence/issues/448)) [@turbolent](https://github.com/turbolent)
- Debug log decode calls ([#585](https://github.com/onflow/cadence/issues/585)) [@turbolent](https://github.com/turbolent)

## 🛠 Improvements

- Make CompositeType.Members field an ordered map ([#581](https://github.com/onflow/cadence/issues/581)) [@SupunS](https://github.com/SupunS)
- Add ordered map for nested types in the checker ([#580](https://github.com/onflow/cadence/issues/580)) [@SupunS](https://github.com/SupunS)
- Use elaborations instead of checkers ([#576](https://github.com/onflow/cadence/issues/576)) [@turbolent](https://github.com/turbolent)
- Check ranges over maps ([#450](https://github.com/onflow/cadence/issues/450)) [@turbolent](https://github.com/turbolent)
- Improve resource info merging ([#606](https://github.com/onflow/cadence/issues/606)) [@turbolent](https://github.com/turbolent)
- Make dictionary entries and composite fields deterministic ([#614](https://github.com/onflow/cadence/issues/614)) [@turbolent](https://github.com/turbolent)
- Write cached items in deterministic order ([#613](https://github.com/onflow/cadence/issues/613)) [@turbolent](https://github.com/turbolent)
- Add tests for KeyString function ([#612](https://github.com/onflow/cadence/issues/612)) [@turbolent](https://github.com/turbolent)
- Use ordered map for field initialization check ([#609](https://github.com/onflow/cadence/issues/609)) [@turbolent](https://github.com/turbolent)
- Use ordered maps for type parameters  ([#608](https://github.com/onflow/cadence/issues/608)) [@turbolent](https://github.com/turbolent)
- Use ordered maps for imported values and types ([#607](https://github.com/onflow/cadence/issues/607)) [@turbolent](https://github.com/turbolent)
- Cache members of composite types and interface types ([#611](https://github.com/onflow/cadence/issues/611)) [@turbolent](https://github.com/turbolent)
- Reuse enc mode and dec mode ([#610](https://github.com/onflow/cadence/issues/610)) [@turbolent](https://github.com/turbolent)
- Make resource tracking deterministic  ([#603](https://github.com/onflow/cadence/issues/603)) [@turbolent](https://github.com/turbolent)
- Make base values and base types deterministic ([#604](https://github.com/onflow/cadence/issues/604)) [@turbolent](https://github.com/turbolent)
- Make elaboration globals deterministic ([#605](https://github.com/onflow/cadence/issues/605)) [@turbolent](https://github.com/turbolent)
- Make activations deterministic ([#601](https://github.com/onflow/cadence/issues/601)) [@turbolent](https://github.com/turbolent)
- Make member set deterministic ([#602](https://github.com/onflow/cadence/issues/602)) [@turbolent](https://github.com/turbolent)
- Make interface sets ordered and thread-safe ([#600](https://github.com/onflow/cadence/issues/600)) [@turbolent](https://github.com/turbolent)
- Improve error reporting in contract update validation ([#597](https://github.com/onflow/cadence/issues/597)) [@SupunS](https://github.com/SupunS)
- Optimize resource tracking ([#592](https://github.com/onflow/cadence/issues/592)) [@turbolent](https://github.com/turbolent)
- Make iteration deterministic ([#596](https://github.com/onflow/cadence/issues/596)) [@turbolent](https://github.com/turbolent)
- Optimize member set by using plain Go maps and parent-child chaining ([#589](https://github.com/onflow/cadence/issues/589)) [@turbolent](https://github.com/turbolent)

## 🐞 Bug Fixes

- Improve Virtual Machine ([#598](https://github.com/onflow/cadence/issues/598)) [@turbolent](https://github.com/turbolent)
- Fix parsing 'from' keyword in imported-identifiers ([#577](https://github.com/onflow/cadence/issues/577)) [@SupunS](https://github.com/SupunS)

## 📖 Documentation

- Minor additions to the docs ([#594](https://github.com/onflow/cadence/issues/594)) [@janezpodhostnik](https://github.com/janezpodhostnik)
- Add compilation to runtime diagram ([#591](https://github.com/onflow/cadence/issues/591)) [@turbolent](https://github.com/turbolent)


[Changes][v0.13.0]


<a name="v0.12.11"></a>
# [v0.12.11](https://github.com/onflow/cadence/releases/tag/v0.12.11) - 15 Feb 2021

## 🐞 Bug Fixes

- Fix `String()` function of `DicitonaryValue` when values are deferred ([#587](https://github.com/onflow/cadence/issues/587))

[Changes][v0.12.11]


<a name="v0.12.10"></a>
# [v0.12.10](https://github.com/onflow/cadence/releases/tag/v0.12.10) - 15 Feb 2021

## 🐞 Bug Fixes

- Revert dictionary key string format for address values ([#586](https://github.com/onflow/cadence/issues/586))

[Changes][v0.12.10]


<a name="v0.12.9"></a>
# [v0.12.9](https://github.com/onflow/cadence/releases/tag/v0.12.9) - 15 Feb 2021

## 🐞 Bug Fixes

- Fix log statement ([1328925](https://github.com/onflow/cadence/commit/1328925a8221d76c41e51f1df81424566262f10c))

[Changes][v0.12.9]


<a name="v0.12.8"></a>
# [v0.12.8](https://github.com/onflow/cadence/releases/tag/v0.12.8) - 15 Feb 2021

## ⭐ Features

- Add an optional callback for encoding prepare function calls and debug log them to the runtime interface ([#584](https://github.com/onflow/cadence/issues/584))


[Changes][v0.12.8]


<a name="v0.12.7"></a>
# [v0.12.7](https://github.com/onflow/cadence/releases/tag/v0.12.7) - 04 Feb 2021

## ⭐ Features

- WebAssembly: Add support for start section ([#430](https://github.com/onflow/cadence/issues/430))
- WebAssembly: Add support for memory exports ([#427](https://github.com/onflow/cadence/issues/427))

## 🛠 Improvements

- Return a dedicated error for encoding an unsupported value ([#583](https://github.com/onflow/cadence/issues/583))
- Update language server to Cadence v0.12.6 and Go SDK v0.14.3 ([#579](https://github.com/onflow/cadence/issues/579))


[Changes][v0.12.7]


<a name="v0.12.6"></a>
# [v0.12.6](https://github.com/onflow/cadence/releases/tag/v0.12.6) - 02 Feb 2021

## 🛠 Improvements

- Optimize activations ([#571](https://github.com/onflow/cadence/issues/571))
- Make occurrences and origins optional ([#570](https://github.com/onflow/cadence/issues/570))
- Update to hamt 37930cf9f7d8, which contains ForEach, optimize activations ([#569](https://github.com/onflow/cadence/issues/569))
- Move global values, global types, and transaction types to elaboration ([#558](https://github.com/onflow/cadence/issues/558))
- Improve AST indices ([#556](https://github.com/onflow/cadence/issues/556))
- Cache programs before checking succeeded, properly wrap returned error ([#557](https://github.com/onflow/cadence/issues/557))

Performance of checking and interpretation has been improved, 
e.g. for `BenchmarkCheckContractInterfaceFungibleTokenConformance`:

| Commit   | Description                                    |  Ops |          Time |
|----------|------------------------------------------------|------:|--------------:|
| 21764d89 | Baseline, v0.12.5                              |   595 | 2018162 ns/op |
| df2ba05d | Update hamt, use ForEach everywhere            |  1821 |  658515 ns/op |
| 6088ce01 | Optional occurrences and origins recording     |  2258 |  530121 ns/op |
| 429a7796 | Optimize activations, replace use of HAMT map  |  3667 |  327368 ns/op |


## ⭐ Features

- Optionally run checker tests concurrently ([#554](https://github.com/onflow/cadence/issues/554))

## 🐞 Bug Fixes

- Fix runtime type of Block.timestamp ([#575](https://github.com/onflow/cadence/issues/575))
- Use correct CommandSubmitTransaction command in entrypoint code lens ([#568](https://github.com/onflow/cadence/issues/568))
- Revert "always find the declared variable ([#408](https://github.com/onflow/cadence/issues/408))" ([#561](https://github.com/onflow/cadence/issues/561))
- Make parameter list thread-safe ([#555](https://github.com/onflow/cadence/issues/555))

## 📖 Documentation

- More developer documentation ([#535](https://github.com/onflow/cadence/issues/535))
- Update the roadmap ([#553](https://github.com/onflow/cadence/issues/553))
- Document how to inject value declarations ([#547](https://github.com/onflow/cadence/issues/547))


[Changes][v0.12.6]


<a name="v0.10.6"></a>
# [v0.10.6](https://github.com/onflow/cadence/releases/tag/v0.10.6) - 30 Jan 2021

## 🛠 Improvements

- Update to hamt 37930cf9f7d8, use `ForEach` instead of `FirstRest` ([#566](https://github.com/onflow/cadence/issues/566))
- Make checker occurrences and origins optional ([#567](https://github.com/onflow/cadence/issues/567))


[Changes][v0.10.6]


<a name="v0.12.5"></a>
# [v0.12.5](https://github.com/onflow/cadence/releases/tag/v0.12.5) - 22 Jan 2021

## ⭐ Features

- WebAssembly: Add support for memory and data sections ([#425](https://github.com/onflow/cadence/issues/425))
- Start of a compiler, with IR and WASM code generator ([#409](https://github.com/onflow/cadence/issues/409))
- Language Server: Parse pragma arguments declarations and show codelenses ([#432](https://github.com/onflow/cadence/issues/432))

## 🛠 Improvements

- Update the parser NPM package ([#544](https://github.com/onflow/cadence/issues/544))
- Update the language server and Monaco client NPM packages ([#545](https://github.com/onflow/cadence/issues/545))
- Interpreter: Always find the declared variable ([#408](https://github.com/onflow/cadence/issues/408))

## 🐞 Bug Fixes

- Fix the export of type values with restricted static types ([#551](https://github.com/onflow/cadence/issues/551))
- Fix nested enum declarations ([#549](https://github.com/onflow/cadence/issues/549))
- Language Server: Don't panic when notifications fail ([#543](https://github.com/onflow/cadence/issues/543))


[Changes][v0.12.5]


<a name="v0.12.4"></a>
# [v0.12.4](https://github.com/onflow/cadence/releases/tag/v0.12.4) - 20 Jan 2021

## ⭐ Features

- Argument list parsing and transaction declaration docstrings ([#528](https://github.com/onflow/cadence/issues/528))
- Add support for name section in WASM binary ([#388](https://github.com/onflow/cadence/issues/388))
- Add more WASM instructions ([#381](https://github.com/onflow/cadence/issues/381))
- Add support for export section in WASM binary writer and reader ([#377](https://github.com/onflow/cadence/issues/377))
- Generate code for instructions ([#372](https://github.com/onflow/cadence/issues/372))

## 🛠 Improvements

- WebAssembly package improvements ([#542](https://github.com/onflow/cadence/issues/542))
- Generate code for instructions ([#372](https://github.com/onflow/cadence/issues/372))

## 🐞 Bug Fixes

- Add support for multiple contracts per account to the language server ([#540](https://github.com/onflow/cadence/issues/540))


[Changes][v0.12.4]


<a name="v0.12.3"></a>
# [v0.12.3](https://github.com/onflow/cadence/releases/tag/v0.12.3) - 14 Jan 2021

## 🛠 Improvements

- Improve the parsing / checking error ([#525](https://github.com/onflow/cadence/issues/525))


[Changes][v0.12.3]


<a name="v0.12.2"></a>
# [v0.12.2](https://github.com/onflow/cadence/releases/tag/v0.12.2) - 13 Jan 2021

## 🛠 Improvements

- Export all checkers ([#524](https://github.com/onflow/cadence/issues/524))

## 📖 Documentation

- Add grammar for Cadence  ([#522](https://github.com/onflow/cadence/issues/522))
- Add developer documentation ([#523](https://github.com/onflow/cadence/issues/523))
- Fix typos in documentation ([#519](https://github.com/onflow/cadence/issues/519))


[Changes][v0.12.2]


<a name="v0.12.1"></a>
# [v0.12.1](https://github.com/onflow/cadence/releases/tag/v0.12.1) - 08 Jan 2021

## 🛠 Improvements

- Improve the contract deployment error message ([#507](https://github.com/onflow/cadence/issues/507))

## 🐞 Bug Fixes

- Gracefully handle type loading failure ([#510](https://github.com/onflow/cadence/issues/510)) 

## 📖 Documentation

- Remove completed items from roadmap ([#515](https://github.com/onflow/cadence/issues/515))
- Add presentation about implementation ([#506](https://github.com/onflow/cadence/issues/506))


[Changes][v0.12.1]


<a name="v0.10.5"></a>
# [v0.10.5](https://github.com/onflow/cadence/releases/tag/v0.10.5) - 08 Jan 2021

## 🛠 Improvements

- Improve fixed-point multiplication and division ([#508](https://github.com/onflow/cadence/issues/508))
- Make AST thread-safe ([#440](https://github.com/onflow/cadence/issues/440))

## 🐞 Bug Fixes

- Gracefully handle type loading failure ([#511](https://github.com/onflow/cadence/issues/511))


[Changes][v0.10.5]


<a name="v0.12.0"></a>
# [v0.12.0](https://github.com/onflow/cadence/releases/tag/v0.12.0) - 15 Dec 2020

## ⭐ Features

- Add `getType` function ([#493](https://github.com/onflow/cadence/issues/493)): It is now possible to get the [run-time type](https://docs.onflow.org/cadence/language/run-time-types/) of a value
- Flush the cache of the storage before querying the used storage amount ([#480](https://github.com/onflow/cadence/issues/480))
- Structured type identifiers ([#477](https://github.com/onflow/cadence/issues/477))
- Allow host environment to predeclare values, predicated on location ([#472](https://github.com/onflow/cadence/issues/472))
- Add a visitor for interpreter values ([#449](https://github.com/onflow/cadence/issues/449))
- Add support for imports in WASM writer and reader ([#368](https://github.com/onflow/cadence/issues/368))
- Add support for coverage reports ([#465](https://github.com/onflow/cadence/issues/465))
- Add storage fields to accounts ([#439](https://github.com/onflow/cadence/issues/439))
- Implement `fmt.Stringer` for `cadence.Value` ([#434](https://github.com/onflow/cadence/issues/434))
- Add a function to parse a literal with a given target type ([#417](https://github.com/onflow/cadence/issues/417))

## 🛠 Improvements

- Extend event parameter types and dictionary key types ([#497](https://github.com/onflow/cadence/issues/497))
- Optimize composite and interface static types ([#489](https://github.com/onflow/cadence/issues/489))
- Optimize composite values ([#488](https://github.com/onflow/cadence/issues/488))
- Fix the export of static types ([#487](https://github.com/onflow/cadence/issues/487))
- Improve error pretty printing ([#481](https://github.com/onflow/cadence/issues/481))
- Improve fixed-point multiplication and division ([#490](https://github.com/onflow/cadence/issues/490))
- Improve error messages for contract deployment name argument checks ([#475](https://github.com/onflow/cadence/issues/475))
- Add a test for decoding a struct with an address location without name ([#469](https://github.com/onflow/cadence/issues/469))
- Refactor address locations, make composite decoding backwards-compatible ([#457](https://github.com/onflow/cadence/issues/457))
- Improve error message when there are constructor argument ([#455](https://github.com/onflow/cadence/issues/455))
- Make AST thread-safe ([#440](https://github.com/onflow/cadence/issues/440))
- Add position information to interpreter errors ([#424](https://github.com/onflow/cadence/issues/424))

## 🐞 Bug Fixes

- Declare new contract's nested values before evaluating initializer ([#504](https://github.com/onflow/cadence/issues/504))
- Infer address location name from type ID for static types ([#468](https://github.com/onflow/cadence/issues/468))
- Fix optional value's type function ([#458](https://github.com/onflow/cadence/issues/458))
- Don't use the cache when deploying or updating account code ([#447](https://github.com/onflow/cadence/issues/447))
- Properly handle unspecified variable kind ([#441](https://github.com/onflow/cadence/issues/441))
- Prevent resource loss in failable downcasts ([#426](https://github.com/onflow/cadence/issues/426))

## 💥 Breaking Changes

This release contains no source-breaking changes for Cadence programs, but the following breaking changes when embedding Cadence:

- Structured type identifiers ([#477](https://github.com/onflow/cadence/issues/477))
- Add error return value to all interface methods ([#470](https://github.com/onflow/cadence/issues/470))

## 📖 Documentation

- Document that references are not storable and suggest using capabilities ([#478](https://github.com/onflow/cadence/issues/478))
- Update the diagram illustrating the architecture of the runtime ([#476](https://github.com/onflow/cadence/issues/476))
- Document the current options for syntax highlighting ([#444](https://github.com/onflow/cadence/issues/444))


[Changes][v0.12.0]


<a name="v0.10.4"></a>
# [v0.10.4](https://github.com/onflow/cadence/releases/tag/v0.10.4) - 09 Dec 2020

## 🛠 Improvements
 
- Allow non-fatal errors for all interface functions ([#494](https://github.com/onflow/cadence/issues/494), [#495](https://github.com/onflow/cadence/issues/495))
- Panic with array index out of bounds error ([#496](https://github.com/onflow/cadence/issues/496))


[Changes][v0.10.4]


<a name="v0.10.3"></a>
# [v0.10.3](https://github.com/onflow/cadence/releases/tag/v0.10.3) - 04 Dec 2020

## ⭐ Features

- Add storage fields to accounts ([#485](https://github.com/onflow/cadence/issues/485))

## 🛠 Improvements

- Flush the cache of the storage before querying the used storage amount ([#486](https://github.com/onflow/cadence/issues/486))


[Changes][v0.10.3]


<a name="v0.11.2"></a>
# [v0.11.2](https://github.com/onflow/cadence/releases/tag/v0.11.2) - 30 Nov 2020

## ⭐ Features

- Extended debug ([#464](https://github.com/onflow/cadence/issues/464))

## 🛠 Improvements

- Refactor address locations, make composite decoding backwards-compatible ([#461](https://github.com/onflow/cadence/issues/461))


[Changes][v0.11.2]


<a name="v0.10.2"></a>
# [v0.10.2](https://github.com/onflow/cadence/releases/tag/v0.10.2) - 23 Nov 2020

## ⭐ Features

- Extended debug ([#463](https://github.com/onflow/cadence/issues/463))

## 🛠 Improvements

- Refactor address locations, make composite decoding backwards-compatible ([#460](https://github.com/onflow/cadence/issues/460))


[Changes][v0.10.2]


<a name="v0.9.3"></a>
# [v0.9.3](https://github.com/onflow/cadence/releases/tag/v0.9.3) - 19 Nov 2020

## ⭐ Features

- Wrap errors to provide additional information ([#451](https://github.com/onflow/cadence/issues/451))


[Changes][v0.9.3]


<a name="v0.11.1"></a>
# [v0.11.1](https://github.com/onflow/cadence/releases/tag/v0.11.1) - 09 Nov 2020

## 🐞 Bug Fixes

- Don't use the cache when deploying or updating account code ([#447](https://github.com/onflow/cadence/issues/447))

[Changes][v0.11.1]


<a name="v0.10.1"></a>
# [v0.10.1](https://github.com/onflow/cadence/releases/tag/v0.10.1) - 06 Nov 2020

## 🐞 Bug Fixes

- Don't use the cache when deploying or updating account code ([#447](https://github.com/onflow/cadence/issues/447))

[Changes][v0.10.1]


<a name="v0.11.0"></a>
# [v0.11.0](https://github.com/onflow/cadence/releases/tag/v0.11.0) - 13 Oct 2020

## 💥 Breaking Changes

### Typed Paths ([#403](https://github.com/onflow/cadence/issues/403))

Paths are now typed. Paths in the storage domain have type `StoragePath`, in the private domain `PrivatePath`, and in the public domain `PublicPath`.  `PrivatePath` and `PublicPath` are subtypes of `CapabilityPath`. Both `StoragePath` and `CapabilityPath` are subtypes of `Path`.

<table>
  <tr>
    <td colspan="3">Path</td>
  </tr>
  <tr>
    <td colspan="2">CapabilityPath</td>
    <td colspan="2" rowspan="2">StoragePath</td>
  </tr>
  <tr>
    <td>PrivatePath</td>
    <td>PublicPath</td>
  </tr>
</table>

### Storage API ([#403](https://github.com/onflow/cadence/issues/403))

With paths being typed, it was possible to make the Storage API type-safer and easier to use: It is now statically checked if the correct type of path is given to a function, instead of at run-time, and therefore capability return types can now be non-optional.

The changes are as follows:

For `PublicAccount`:

- old: `fun getCapability<T>(_ path: Path): Capability<T>?` <br/>
  new: `fun getCapability<T>(_ path: PublicPath): Capability<T>`

- old: `fun getLinkTarget(_ path: Path): Path?` <br />
  new: `fun getLinkTarget(_ path: CapabilityPath): Path?`

For `AuthAccount`:

- old: `fun save<T>(_ value: T, to: Path)` <br />
  new: `fun save<T>(_ value: T, to: StoragePath)`

- old: `fun load<T>(from: Path): T?` <br />
  new: `fun load<T>(from: StoragePath): T?`

- old: `fun copy<T: AnyStruct>(from: Path): T?` <br />
  new: `fun copy<T: AnyStruct>(from: StoragePath): T?`

- old: `fun borrow<T: &Any>(from: Path): T?` <br />
  new: `fun borrow<T: &Any>(from: StoragePath): T?`

- old: `fun link<T: &Any>(_ newCapabilityPath: Path, target: Path): Capability<T>?` <br />
  new: `fun link<T: &Any>(_ newCapabilityPath: CapabilityPath, target: Path): Capability<T>?`

- old: `fun getCapability<T>(_ path: Path): Capability<T>?` <br/>
  new: `fun getCapability<T>(_ path: CapabilityPath): Capability<T>`

- old: `fun getLinkTarget(_ path: Path): Path?` <br />
  new: `fun getLinkTarget(_ path: CapabilityPath): Path?`

- old: `fun unlink(_ path: Path)` <br />
  new: `fun unlink(_ path: CapabilityPath)`

## ⭐ Features

- Add a hash function to the crypto contract ([#379](https://github.com/onflow/cadence/issues/379))
- Added npm packages for components of Cadence. This eases the development of developer tools for Cadence:

  - `cadence-language-server`: The Cadence Language Server
  - `monaco-languageclient-cadence`: Language Server Protocol client for the the Monaco editor
  - `cadence-parser`: The Cadence parser

  In addition, there are also examples for the [language server](https://github.com/onflow/cadence/tree/master/npm-packages/cadence-language-server-demo) and the [parser](https://github.com/onflow/cadence/tree/master/npm-packages/cadence-parser-demo) that demonstrate the use of the packages.

- Add a command to the language server that allows getting the entry point (transaction or script) parameters ([#406](https://github.com/onflow/cadence/issues/406))

## 🛠 Improvements

- Allow references to be returned from from scripts ([#400](https://github.com/onflow/cadence/issues/400))
- Panic with a dedicated error for out of bounds array index ([#396](https://github.com/onflow/cadence/issues/396))

## 📖 Documentation

- Document resource identifiers ([#394](https://github.com/onflow/cadence/issues/394))
- Document iteration over dictionary entries ([#399](https://github.com/onflow/cadence/issues/399))

## 📦 Dependencies

- The changes to the [CBOR library](https://github.com/fxamacker/cbor) have been [merged](https://github.com/fxamacker/cbor/pull/249), so [the `replace` statement that was necessary in the last release](https://github.com/onflow/cadence/releases/tag/v0.10.0) must be removed.


[Changes][v0.11.0]


<a name="v0.9.2"></a>
# [v0.9.2](https://github.com/onflow/cadence/releases/tag/v0.9.2) - 06 Oct 2020

## 🛠️ Improvements

- Panic with a dedicated error for array out of bounds access ([#396](https://github.com/onflow/cadence/issues/396))


[Changes][v0.9.2]


<a name="v0.10.0"></a>
# [v0.10.0](https://github.com/onflow/cadence/releases/tag/v0.10.0) - 01 Oct 2020

## 💥 Breaking Changes

### Contract Deployment

This release adds support for deploying multiple contracts per account.
The API for deploying has changed:

- The functions `AuthAccount.setCode` and `AuthAccount.unsafeNotInitializingSetCode` were removed ([#390](https://github.com/onflow/cadence/issues/390)).
- A [new contract management API](https://docs.onflow.org/cadence/language/contracts/#deploying-updating-and-removing-contracts) has been added, which allows adding, updating, and removing multiple contracts per account ([#333](https://github.com/onflow/cadence/issues/333), [#352](https://github.com/onflow/cadence/issues/352)).

  See the [updated documentation](https://docs.onflow.org/cadence/language/contracts/#deploying-updating-and-removing-contracts) for details and examples.

## ⭐ Features

### Enumerations

This release adds support for enumerations ([#344](https://github.com/onflow/cadence/issues/344)).

For example, a days-of-the-week enum can be declared though:

```swift
enum Day: UInt8 {
    case Monday
    case Tuesday
    case Wednesday
    case Thursday
    case Friday
    case Saturday
    case Sunday
}
```

See the [documentation](https://docs.onflow.org/cadence/language/enumerations/) for further details and examples.

### Switch Statement

This release adds support for switch statements ([#365](https://github.com/onflow/cadence/issues/365)).

```swift
fun describe(number: Int): String {
    switch number {
    case 1:
        return "one"
    case 2:
        return "two"
    default:
        return "other"
    }
}
```

See the [documentation](https://docs.onflow.org/cadence/language/control-flow/#switch) for further details and examples.

### Code Formatter

Development of a code formatter has started, in form of a plugin for Prettier ([#348](https://github.com/onflow/cadence/issues/348)).
If you would like to contribute, please let us know!

## 🛠 Improvements

- Limitations on data structure complexity are now enforced when decoding and also when encoding storage data, e.g number of array elements, number of dictionary entries, etc. ([#370](https://github.com/onflow/cadence/issues/370))
- Using the force-unwrap operator on a non-optional is no longer an error, but a hint is reported suggesting the removal of the unnecessary operation ([#347](https://github.com/onflow/cadence/issues/347))
- Language Server: The features requiring a Flow client connection are now optional. This allows using the language server in editors other than Visual Studio Code, such as Emacs, Vim, etc. ([#303](https://github.com/onflow/cadence/issues/303))

## 🐞 Bug Fixes

- Fixed the encoding of bignums in storage ([#370](https://github.com/onflow/cadence/issues/370))
- Fixed the timing of recording composite-type explicit-conformances ([#356](https://github.com/onflow/cadence/issues/356))
- Added support for exporting and JSON encoding/decoding of type values and capabilities ([#374](https://github.com/onflow/cadence/issues/374))
- Added support for exporting/importing and JSON encoding/decoding of path values ([#319](https://github.com/onflow/cadence/issues/319))
- Fixed the handling of empty integer literals in the checker ([#354](https://github.com/onflow/cadence/issues/354))
- Fixed the non-storable fields error ([#350](https://github.com/onflow/cadence/issues/350))

## 📖 Documentation

- Added a [roadmap describing planned features and ideas](https://github.com/onflow/cadence/blob/master/ROADMAP.md) ([#367](https://github.com/onflow/cadence/issues/367))
- Various documentation improvements ([#385](https://github.com/onflow/cadence/issues/385)). Thanks to [@andrejtokarcik](https://github.com/andrejtokarcik) for contributing this!

## 📦 Dependencies

This release depends on a fork of the [CBOR library](github.com/fxamacker/cbor/v2) that Cadence uses.
When importing Cadence in another Go codebase, add the following statement to the `go.mod` file:

```
replace github.com/fxamacker/cbor/v2 => github.com/turbolent/cbor/v2 v2.2.1-0.20200911003300-cac23af49154
```

[Changes][v0.10.0]


<a name="v0.9.0"></a>
# [v0.9.0](https://github.com/onflow/cadence/releases/tag/v0.9.0) - 23 Sep 2020

## 🛠 Improvements

- Add struct as supported event parameter type ([#364](https://github.com/onflow/cadence/issues/364))
- Make recursive types storable ([#329](https://github.com/onflow/cadence/issues/329)) 
- Suggest replacing number conversion function calls ([#346](https://github.com/onflow/cadence/issues/346))

## ⭐ Features

- Add `map` function for optionals ([#299](https://github.com/onflow/cadence/issues/299))

## 💥 Breaking Changes

- Require an explicit type annotation for variable declarations with empty literal expressions ([#345](https://github.com/onflow/cadence/issues/345))

## 🐞 Bug Fixes

- Fix parsing of right bitwise shift and function invocation where type argument is instantiation ([#378](https://github.com/onflow/cadence/issues/378))

## 📖 Documentation

- Refactor the documentation ([#380](https://github.com/onflow/cadence/issues/380)). 

  The new documentation is more accessible and available at https://docs.onflow.org/cadence/language


[Changes][v0.9.0]


<a name="v0.8.2"></a>
# [v0.8.2](https://github.com/onflow/cadence/releases/tag/v0.8.2) - 28 Aug 2020

## 🐞 Bug Fixes

- Copy values on return ([#355](https://github.com/onflow/cadence/issues/355))


[Changes][v0.8.2]


<a name="v0.8.1"></a>
# [v0.8.1](https://github.com/onflow/cadence/releases/tag/v0.8.1) - 25 Aug 2020

## 🐞 Bug Fixes

- Validate script argument count ([#316](https://github.com/onflow/cadence/issues/316))


[Changes][v0.8.1]


<a name="v0.8.0"></a>
# [v0.8.0](https://github.com/onflow/cadence/releases/tag/v0.8.0) - 10 Aug 2020

This release focuses on improvements, bug fixes, and bringing the documentation up-to-date.

## 🛠 Improvements

- Improved support on Windows: Treat carriage returns as space ([#304](https://github.com/onflow/cadence/issues/304))
- Improved JSON marshalling for more AST elements ([#292](https://github.com/onflow/cadence/issues/292), [#286](https://github.com/onflow/cadence/issues/286))
- Recursively check if assignment target expression is valid, don't copy returned values ([#288](https://github.com/onflow/cadence/issues/288))
- Consider existing prefix when suggesting completion items ([#285](https://github.com/onflow/cadence/issues/285))
- Include available declarations in "not declared" error ([#280](https://github.com/onflow/cadence/issues/280))
- Enforce the requirement for types to be storable in more places:
  - Transaction/script parameter types ([#305](https://github.com/onflow/cadence/issues/305))
  - Script return type ([#291](https://github.com/onflow/cadence/issues/291))
  - Arguments passed to the account `load` and `store` functions ([#251](https://github.com/onflow/cadence/issues/251))

  Previously, using non-storable failed at run-time, now the programs are rejected statically.

## ⭐ Features

- Add a version constant ([#289](https://github.com/onflow/cadence/issues/289))
- Language Server: Include error notes as related information in diagnostic ([#276](https://github.com/onflow/cadence/issues/276))

## 🐞 Bug Fixes

- Don't return an error when a location type is not supported ([#290](https://github.com/onflow/cadence/issues/290))
- Handle incomplete types in checker ([#284](https://github.com/onflow/cadence/issues/284))
- Fix the suggestion in the error when an interface is used as a type ([#281](https://github.com/onflow/cadence/issues/281))

## 📖 Documentation

- Brought the documentation up-to-date, include all new language features and API changes ([#275](https://github.com/onflow/cadence/issues/275), [#277](https://github.com/onflow/cadence/issues/277), [#279](https://github.com/onflow/cadence/issues/279))

## 💥 Breaking Changes

- Arguments passed to `cadence.Value#WithType` are now typed as pointers (`*cadence.Type`) rather than values (`cadence.Type`)

  Example:

  ```go
  myEventType := cadence.EventType{...})

  // this 👇 
  myEvent := cadence.NewEvent(...).WithType(myEventType)

  // changes to this 👇 
  myEvent := cadence.NewEvent(...).WithType(&myEventType)
  ```


[Changes][v0.8.0]


<a name="v0.7.0"></a>
# [v0.7.0](https://github.com/onflow/cadence/releases/tag/v0.7.0) - 05 Aug 2020

This release contains a lot of improvements to the language server, which improve the development experience in the Visual Studio Code extension, and will also soon be integrated into the Flow Playground.

## ⭐ Features

- Added member completion to the language server ([#257](https://github.com/onflow/cadence/issues/257))
  - Insert the argument list when completing functions ([#268](https://github.com/onflow/cadence/issues/268))
  - Offer keyword snippets ([#268](https://github.com/onflow/cadence/issues/268))
- Enabled running the language server in the browser by compiling it to WebAssembly ([#242](https://github.com/onflow/cadence/issues/242))
- Added support for docstrings to the parser ([#246](https://github.com/onflow/cadence/issues/246), [#255](https://github.com/onflow/cadence/issues/255))
- Added support for pragmas to the parser ([#239](https://github.com/onflow/cadence/issues/239))
- Started a source compatibility suite ([#226](https://github.com/onflow/cadence/issues/226), [#270](https://github.com/onflow/cadence/issues/270)).

  Cadence will be regularly checked for compatibility with existing code to ensure it does not regress in source compatibility and performance. As a start, the [Fungible Token standard](https://github.com/onflow/flow-ft) and [Non-Fungible Token standard](https://github.com/onflow/flow-ft) repositories are checked.

  Consider requesting your repository to be included in the compatibility suite!
- Added a simple minifier ([#252](https://github.com/onflow/cadence/issues/252))

## 🛠 Improvements

- Made embedding the language server easier ([#274](https://github.com/onflow/cadence/issues/274), [#262](https://github.com/onflow/cadence/issues/262))
- Extended the JSON serialization for Cadence values and types ([#260](https://github.com/onflow/cadence/issues/260), [#264](https://github.com/onflow/cadence/issues/264), [#265](https://github.com/onflow/cadence/issues/265), [#266](https://github.com/onflow/cadence/issues/266), [#271](https://github.com/onflow/cadence/issues/271), [#273](https://github.com/onflow/cadence/issues/273)) 
- Improved conformance checking ([#245](https://github.com/onflow/cadence/issues/245)). Made return types covariant, i.e. allow subtypes for return types of functions to satisfy interface conformance.


## 💥 Breaking Changes

- Made strings immutable ([#269](https://github.com/onflow/cadence/issues/269))
- Moved the Visual Studio Code extension to a new repository: <https://github.com/onflow/vscode-flow> ([#244](https://github.com/onflow/cadence/issues/244))
- Added a magic prefix to stored data ([#236](https://github.com/onflow/cadence/issues/236)). 
  This allows versioning the data. Existing data without a prefix is migrated automatically
- Changed the order in which fields in values and types are exported to declaration order ([#272](https://github.com/onflow/cadence/issues/272))

## 🐞 Bug Fixes

- Removed the service account from the list of usable accounts in the language server ([#261](https://github.com/onflow/cadence/issues/261)) 
- Fixed a parsing ambiguity for return types ([#267](https://github.com/onflow/cadence/issues/267))
- Fixed non-composite reference subtyping ([#253](https://github.com/onflow/cadence/issues/253))

## 🧰 Maintenance

- Removed the old parser ([#249](https://github.com/onflow/cadence/issues/249))

[Changes][v0.7.0]


<a name="v0.6.0"></a>
# [v0.6.0](https://github.com/onflow/cadence/releases/tag/v0.6.0) - 14 Jul 2020

This is a small release with some bug fixes, internal improvements, and one breaking change for code that embeds Cadence. 

## 💥 Breaking Changes

- Removed the unused `controller` parameter from the storage interface methods:

  - `func GetValue(owner, controller, key []byte) (value []byte, err error)` 
    → `func GetValue(owner, key []byte) (value []byte, err error)`

  - `SetValue(owner, controller, key, value []byte) (err error)`
    → `SetValue(owner, key, value []byte) (err error)`

  - `ValueExists(owner, controller, key []byte) (exists bool, err error)`
    → `ValueExists(owner, key []byte) (exists bool, err error)`

  This is only a breaking change in the implementation of Cadence, i.e for code that embeds Cadence, not in the Cadence language, i.e. for users of Cadence.

## ⭐ Features

- Added an optional callback for writes with high-level values to the interface:

  ```go
  type HighLevelStorage interface {

	  // HighLevelStorageEnabled should return true
	  // if the functions of HighLevelStorage should be called,
	  // e.g. SetCadenceValue
	  HighLevelStorageEnabled() bool

	  // SetCadenceValue sets a value for the given key in the storage, owned by the given account.
	  SetCadenceValue(owner Address, key string, value cadence.Value) (err error)
  }
  ```

  This is a feature in the implementation of Cadence, i.e for code that embeds Cadence, not in the Cadence language, i.e. for users of Cadence.

## 🛠 Improvements

- Don't report an error for a restriction with an invalid type
- Record the occurrences of types. This enables the "Go to definition" feature for types in the language server
- Parse member expressions without a name. This allows the checker to run. This will enable adding code completion support or members in the future.

## 🐞 Bug Fixes

- Fixed a crash when checking the invocation of a function on an undeclared variable
- Fixed handling of functions in composite values in the JSON-CDC encoding
- Fixed a potential stack overflow when checking member storability


[Changes][v0.6.0]


<a name="v0.5.1"></a>
# [v0.5.1](https://github.com/onflow/cadence/releases/tag/v0.5.1) - 10 Jul 2020

## 🐞 Bug Fixes

- Fix borrowing of capability without type argument

[Changes][v0.5.1]


<a name="v0.5.0"></a>
# [v0.5.0](https://github.com/onflow/cadence/releases/tag/v0.5.0) - 07 Jul 2020

## ⭐ Features and Improvements

### Crypto

A new built-in contract `Crypto` was added for performing cryptographic operations.

The contract can be imported using `import Crypto`.

This first iteration provides support for validating signatures, with an API that supports multiple signatures and weighted keys.

For example, to verify two signatures with equal weights for some signed data:

```swift
let keyList = Crypto.KeyList()

let publicKeyA = Crypto.PublicKey(
    publicKey:
        "db04940e18ec414664ccfd31d5d2d4ece3985acb8cb17a2025b2f1673427267968e52e2bbf3599059649d4b2cce98fdb8a3048e68abf5abe3e710129e90696ca".decodeHex(),
    signatureAlgorithm: Crypto.ECDSA_P256
)
keyList.add(
    publicKeyA,
    hashAlgorithm: Crypto.SHA3_256,
    weight: 0.5
)

let publicKeyB = Crypto.PublicKey(
    publicKey:
        "df9609ee588dd4a6f7789df8d56f03f545d4516f0c99b200d73b9a3afafc14de5d21a4fc7a2a2015719dc95c9e756cfa44f2a445151aaf42479e7120d83df956".decodeHex(),
    signatureAlgorithm: Crypto.ECDSA_P256
)
keyList.add(
    publicKeyB,
    hashAlgorithm: Crypto.SHA3_256,
    weight: 0.5
)

let signatureSet = [
    Crypto.KeyListSignature(
        keyIndex: 0,
        signature:
            "8870a8cbe6f44932ba59e0d15a706214cc4ad2538deb12c0cf718d86f32c47765462a92ce2da15d4a29eb4e2b6fa05d08c7db5d5b2a2cd8c2cb98ded73da31f6".decodeHex()
    ),
    Crypto.KeyListSignature(
        keyIndex: 1,
        signature:
            "bbdc5591c3f937a730d4f6c0a6fde61a0a6ceaa531ccb367c3559335ab9734f4f2b9da8adbe371f1f7da913b5a3fdd96a871e04f078928ca89a83d841c72fadf".decodeHex()
    )
]

// "foo", encoded as UTF-8, in hex representation
let signedData = "666f6f".decodeHex()

let isValid = keyList.isValid(
    signatureSet: signatureSet,
    signedData: signedData
)
```

### Run-time Types

The type `Type` was added to represent types at run-time.

To create a type value, use the constructor function `Type<T>()`, which accepts the static type as a type argument.

This is similar to e.g. `T.self` in Swift, `T::class` in Kotlin, and `T.class` in Java.

For example, to represent the type `Int` at run-time:

```swift
let intType: Type = Type<Int>()
```

This works for both built-in and user-defined types. For example, to get the type value for a resource:

```swift
resource Collectible {}

let collectibleType = Type<@Collectible>()
```

The function `fun isInstance(_ type: Type): Bool` can be used to check if a value has a certain type:

```swift
let collectible <- create Collectible()
let collectibleType = Type<@Collectible>()
let result = collectible.isInstance(collectibleType)
```

For example, this allows implementing a marketplace sale resource:

```swift
pub resource SimpleSale {

    pub var objectForSale: @AnyResource?
    pub let priceForObject: UFix64
    pub let requiredCurrency: Type
    pub let paymentReceiver: Capability<&{FungibleToken.Receiver}>

    init(
        objectForSale: @AnyResource,
        priceForObject: UFix64,
        requiredCurrency: Type,
        paymentReceiver: Capability<&{FungibleToken.Receiver}>
    ) {
        self.objectForSale <- objectForSale
        self.priceForObject = priceForObject
        self.requiredCurrency = requiredCurrency
        self.paymentReceiver = paymentReceiver
    }

    destroy() {
        destroy self.objectForSale
    }

    pub fun buyObject(purchaseAmount: @FungibleToken.Vault): @AnyResource {
        pre {
            self.objectForSale != nil
            purchaseAmount.balance >= self.priceForObject
            purchaseAmount.isInstance(self.requiredCurrency)
        }

        let receiver = self.paymentReceiver.borrow()
            ?? panic("failed to borrow payment receiver capability")

        receiver.deposit(from: <-purchaseAmount)
        let objectForSale <- self.objectForSale <- nil
        return <-objectForSale
    }
}
```

### New Parser

The existing parser was replaced by a new implementation, completely written from scratch, producing the same result as the old parser.

It is significantly faster than the old parser. For example, the following benchmark shows the difference for parsing all fungible and non-fungible token contracts and example transactions:

```sh
$ go run ./cmd/parse -bench ../../flow-{n,}ft/{contracts,transactions}/*
flow-nft/contracts/ExampleNFT.cdc:          [old]        9  111116110 ns/op
flow-nft/contracts/ExampleNFT.cdc:          [new]     2712     463483 ns/op
flow-nft/contracts/NonFungibleToken.cdc:    [old]      393    3097172 ns/op
flow-nft/contracts/NonFungibleToken.cdc:    [new]     3489     348496 ns/op
flow-nft/transactions/mint_nft.cdc:         [old]      700    1574730 ns/op
flow-nft/transactions/mint_nft.cdc:         [new]    12770      94070 ns/op
flow-nft/transactions/read_nft_data.cdc:    [old]      994    1222887 ns/op
flow-nft/transactions/read_nft_data.cdc:    [new]    15242      79295 ns/op
flow-nft/transactions/setup_account.cdc:    [old]     1281     879751 ns/op
flow-nft/transactions/setup_account.cdc:    [new]    16675      71759 ns/op
flow-nft/transactions/transfer_nft.cdc:     [old]       72   16417568 ns/op
flow-nft/transactions/transfer_nft.cdc:     [new]    10000     109380 ns/op
flow-ft/contracts/CustodialDeposit.cdc:     [old]       18   64938763 ns/op
flow-ft/contracts/CustodialDeposit.cdc:     [new]     3482     354662 ns/op
flow-ft/contracts/FlowToken.cdc:            [old]        7  177111544 ns/op
flow-ft/contracts/FlowToken.cdc:            [new]     1920     640557 ns/op
flow-ft/contracts/FungibleToken.cdc:        [old]      232    5324962 ns/op
flow-ft/contracts/FungibleToken.cdc:        [new]     2947     419529 ns/op
flow-ft/contracts/TokenForwarding.cdc:      [old]       44   25136749 ns/op
flow-ft/contracts/TokenForwarding.cdc:      [new]     7183     172917 ns/op
flow-ft/transactions/burn_tokens.cdc:       [old]       37   31475393 ns/op
flow-ft/transactions/burn_tokens.cdc:       [new]    11361     105932 ns/op
flow-ft/transactions/create_forwarder.cdc:  [old]      733    1636347 ns/op
flow-ft/transactions/create_forwarder.cdc:  [new]     8127     147520 ns/op
flow-ft/transactions/create_minter.cdc:     [old]     1306     923201 ns/op
flow-ft/transactions/create_minter.cdc:     [new]    15240      77666 ns/op
flow-ft/transactions/custodial_deposit.cdc: [old]       69   16504795 ns/op
flow-ft/transactions/custodial_deposit.cdc: [new]     7940     144228 ns/op
flow-ft/transactions/get_balance.cdc:       [old]     1094    1111272 ns/op
flow-ft/transactions/get_balance.cdc:       [new]    18741      65745 ns/op
flow-ft/transactions/get_supply.cdc:        [old]     1392     740989 ns/op
flow-ft/transactions/get_supply.cdc:        [new]    46008      26719 ns/op
flow-ft/transactions/mint_tokens.cdc:       [old]       72   17435128 ns/op
flow-ft/transactions/mint_tokens.cdc:       [new]     8841     124117 ns/op
flow-ft/transactions/setup_account.cdc:     [old]     1219    1089357 ns/op
flow-ft/transactions/setup_account.cdc:     [new]    13797      84948 ns/op
flow-ft/transactions/transfer_tokens.cdc:   [old]       74   17011751 ns/op
flow-ft/transactions/transfer_tokens.cdc:   [new]     9578     125829 ns/op
```

The new parser also provides better error messages and will allow us to provide even better error messages in the future.

The new parser is enabled by default – if you discover any problems or regressions, please report them!

### Typed Capabilities

Capabilities now accept an optional type argument, the reference type the capability should be borrowed as.

If a type argument is provided, it will be used for `borrow` and `check` calls, so there is no need to provide a type argument for the calls anymore.

The function `getCapability` now also accepts an optional type argument, and returns a typed capability if one is provided.

For example, the following two functions have the same behaviour:

```swift
fun cap1(): &Something? {
  // The type annotation for `cap` is only added for demonstration purposes,
  // it can also be omitted, because it can be inferred from the value
  let cap: Capability = getAccount(0x1).getCapability(/public/something)!
  return cap.borrow<&Something>()
}

fun cap2(): &Something? {
  // The type annotation for `cap` is only added for demonstration purposes,
  // it can also be omitted, because it can be inferred from the value
  let cap: Capability<&Something> =
      getAccount(0x1).getCapability<&Something>(/public/something)!
  return cap.borrow()
}
```

Prefer a typed capability over a non-typed one to reduce uses / borrows to a simple `.borrow()`, instead of having to repeatedly mention the borrowed type.

### Parameters for Scripts

Just like transactions, the `main` functions of scripts can now have parameters.

For example, a script that can be passed an integer and a string, and which logs both values, can be written as:

```swift
pub fun main(x: Int, y: String) {
    log(x)
    log(y)
}
```

### Standard Library

The function `fun toString(): String` was added to all number types and addresses. It returns the textual representation of the type. A future version of Cadence will add this function to more types.

The function `fun toBytes(): [UInt8]` was added to `Address`. It returns the byte representation of the address.

The function `fun toBigEndianBytes(): [UInt8]` was added to all number types. It returns the big-endian byte representation of the number value.

## 🐞 Bug Fixes

- Fixed the checking of return statements:
  A missing return value is now properly reported
- Fixed the checking of function invocations:
  Resources are temporarily moved into the invoked function
- Disabled caching of top-level programs (transactions, scripts)
- Fixed the comparison of optional values
- Fixed parsing of unpadded fixed point numbers
  in the JSON encoding of Cadence values (JSON-CDC)

## 💥 Breaking Changes

### Field Types

Fields which have a non-storable type are now rejected:

Non-storable types are:

- Functions (e.g. `((Int): Bool)`)
- Accounts (`AuthAccount` / `PublicAccount`)
- Transactions
- `Void`

A future version will also make references non-storable. Instead of storing a reference, store a capability and borrow it to acquire a reference.
### Byte Arrays

Cadence now represents all byte arrays as `[UInt8]` rather than `[Int]`. This affects the following functions:

- `String.decodeHex(): [Int]` => `String.decodeHex(): [UInt8]`
- `AuthAccount.addPublicKey(publicKey: [Int])` => `AuthAccount.addPublicKey(publicKey: [UInt8])`
- `AuthAccount.setCode(code: [Int])` => `AuthAccount.setCode(code: [UInt8])`

However, array literals such as the following will still be inferred as `[Int]`, meaning that they can't be used directly:

```swift
myAccount.addPublicKey([1, 2, 3])
```

Consider using `String.decodeHex()` for now until type inference has been improved.


[Changes][v0.5.0]


<a name="v0.4.0"></a>
# [v0.4.0](https://github.com/onflow/cadence/releases/tag/v0.4.0) - 02 Jun 2020

## 💥 Breaking Changes

- The `AuthAccount` constructor now requires a `payer` parameter, from which the account creation fee is deducted. The `publicKeys` and `code` parameters were removed.

## ⭐ Features

- Added the function `unsafeNotInitializingSetCode` to `AuthAccount` which, like `setCode`, updates the account's code but does not create a new contract object or invoke its initializer. This function can be used to upgrade account code without overwriting the existing stored contract object.

  ⚠️ This is potentially unsafe because no data migration is performed. For example, changing a field's type results in undefined behaviour (i.e. could lead to the execution being aborted).

## 🐞 Bug Fixes

- Fixed variable shadowing for-in identifier to shadow outer scope ([#72](https://github.com/onflow/cadence/issues/72))
- Fixed order of declaring conformances and usages
- Fixed detection of resource losses in invocation of optional chaining result


[Changes][v0.4.0]


<a name="v0.3.0"></a>
# [v0.3.0](https://github.com/onflow/cadence/releases/tag/v0.3.0) - 26 May 2020

## 💥 Breaking Changes

- Dictionary values which are resources are now stored in separate storage keys. This is transparent to programs, so is not breaking on a program level, but on the integration level.
- The state encoding was switched from Gob to CBOR. This is transparent to programs, so is not breaking on a program level, but on the integration level.
- The size of account addresses was reduced from 160 bits to 64 bits 

## ⭐ Features

- Added bitwise operations
- Added support for caching of program ASTs
- Added metrics
- Block info can be queried now

## 🛠 Improvements

- Unnecessary writes are now avoided by tracking modifications
- Improved access modifier checks
- Improved the REPL
- Decode and validate transaction arguments
- Work on a new parser has started and is almost complete

## 🐞 Bug Fixes

- Fixed account code updated 
- Fixed the destruction of arrays
- Fixed type requirements
- Implemented modulo for `[U]Fix64`, fix modulo with negative operands
- Implemented division for `[U]Fix64`
- Fixed the `flow.AccountCodeUpdated` event
- Fixed number conversion
- Fixed function return checking 


[Changes][v0.3.0]


[v1.0.0-preview.52]: https://github.com/onflow/cadence/compare/v1.0.0-preview.51...v1.0.0-preview.52
[v1.0.0-preview.51]: https://github.com/onflow/cadence/compare/v1.0.0-preview.50...v1.0.0-preview.51
[v1.0.0-preview.50]: https://github.com/onflow/cadence/compare/v1.0.0-preview.49...v1.0.0-preview.50
[v1.0.0-preview.49]: https://github.com/onflow/cadence/compare/v1.0.0-preview.48...v1.0.0-preview.49
[v1.0.0-preview.48]: https://github.com/onflow/cadence/compare/v1.0.0-preview.47...v1.0.0-preview.48
[v1.0.0-preview.47]: https://github.com/onflow/cadence/compare/v1.0.0-preview.46...v1.0.0-preview.47
[v1.0.0-preview.46]: https://github.com/onflow/cadence/compare/v1.0.0-preview.45...v1.0.0-preview.46
[v1.0.0-preview.45]: https://github.com/onflow/cadence/compare/v1.0.0-preview.44...v1.0.0-preview.45
[v1.0.0-preview.44]: https://github.com/onflow/cadence/compare/v1.0.0-preview.43...v1.0.0-preview.44
[v1.0.0-preview.43]: https://github.com/onflow/cadence/compare/v1.0.0-preview.42...v1.0.0-preview.43
[v1.0.0-preview.42]: https://github.com/onflow/cadence/compare/v1.0.0-preview.41...v1.0.0-preview.42
[v1.0.0-preview.41]: https://github.com/onflow/cadence/compare/v1.0.0-preview.40...v1.0.0-preview.41
[v1.0.0-preview.40]: https://github.com/onflow/cadence/compare/v1.0.0-preview.39...v1.0.0-preview.40
[v1.0.0-preview.39]: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.39...v1.0.0-preview.39
[v1.0.0-preview-atree-register-inlining.39]: https://github.com/onflow/cadence/compare/v1.0.0-preview.38...v1.0.0-preview-atree-register-inlining.39
[v1.0.0-preview.38]: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.38...v1.0.0-preview.38
[v1.0.0-preview-atree-register-inlining.38]: https://github.com/onflow/cadence/compare/v1.0.0-preview.37...v1.0.0-preview-atree-register-inlining.38
[v1.0.0-preview.37]: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.37...v1.0.0-preview.37
[v1.0.0-preview-atree-register-inlining.37]: https://github.com/onflow/cadence/compare/v1.0.0-preview.36...v1.0.0-preview-atree-register-inlining.37
[v1.0.0-preview.36]: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.36...v1.0.0-preview.36
[v1.0.0-preview-atree-register-inlining.36]: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.35...v1.0.0-preview-atree-register-inlining.36
[v1.0.0-preview-atree-register-inlining.35]: https://github.com/onflow/cadence/compare/v1.0.0-preview.35...v1.0.0-preview-atree-register-inlining.35
[v1.0.0-preview.35]: https://github.com/onflow/cadence/compare/v0.42.12...v1.0.0-preview.35
[v0.42.12]: https://github.com/onflow/cadence/compare/v1.0.0-preview.34...v0.42.12
[v1.0.0-preview.34]: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.34...v1.0.0-preview.34
[v1.0.0-preview-atree-register-inlining.34]: https://github.com/onflow/cadence/compare/v1.0.0-preview.33...v1.0.0-preview-atree-register-inlining.34
[v1.0.0-preview.33]: https://github.com/onflow/cadence/compare/v1.0.0-preview.32...v1.0.0-preview.33
[v1.0.0-preview.32]: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.32...v1.0.0-preview.32
[v1.0.0-preview-atree-register-inlining.32]: https://github.com/onflow/cadence/compare/v1.0.0-preview.31...v1.0.0-preview-atree-register-inlining.32
[v1.0.0-preview.31]: https://github.com/onflow/cadence/compare/v1.0.0-preview.30...v1.0.0-preview.31
[v1.0.0-preview.30]: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.30...v1.0.0-preview.30
[v1.0.0-preview-atree-register-inlining.30]: https://github.com/onflow/cadence/compare/v1.0.0-preview.29...v1.0.0-preview-atree-register-inlining.30
[v1.0.0-preview.29]: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.29...v1.0.0-preview.29
[v1.0.0-preview-atree-register-inlining.29]: https://github.com/onflow/cadence/compare/v1.0.0-preview.28...v1.0.0-preview-atree-register-inlining.29
[v1.0.0-preview.28]: https://github.com/onflow/cadence/compare/v1.0.0-preview.27...v1.0.0-preview.28
[v1.0.0-preview.27]: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.28...v1.0.0-preview.27
[v1.0.0-preview-atree-register-inlining.28]: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.27...v1.0.0-preview-atree-register-inlining.28
[v1.0.0-preview-atree-register-inlining.27]: https://github.com/onflow/cadence/compare/v0.42.11...v1.0.0-preview-atree-register-inlining.27
[v0.42.11]: https://github.com/onflow/cadence/compare/v1.0.0-preview.26...v0.42.11
[v1.0.0-preview.26]: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.26...v1.0.0-preview.26
[v1.0.0-preview-atree-register-inlining.26]: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.25...v1.0.0-preview-atree-register-inlining.26
[v1.0.0-preview-atree-register-inlining.25]: https://github.com/onflow/cadence/compare/v1.0.0-preview.25...v1.0.0-preview-atree-register-inlining.25
[v1.0.0-preview.25]: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.24...v1.0.0-preview.25
[v1.0.0-preview-atree-register-inlining.24]: https://github.com/onflow/cadence/compare/v1.0.0-preview.24...v1.0.0-preview-atree-register-inlining.24
[v1.0.0-preview.24]: https://github.com/onflow/cadence/compare/v1.0.0-preview.23...v1.0.0-preview.24
[v1.0.0-preview.23]: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.23...v1.0.0-preview.23
[v1.0.0-preview-atree-register-inlining.23]: https://github.com/onflow/cadence/compare/v1.0.0-preview.22...v1.0.0-preview-atree-register-inlining.23
[v1.0.0-preview.22]: https://github.com/onflow/cadence/compare/v1.0.0-preview-atree-register-inlining.22...v1.0.0-preview.22
[v1.0.0-preview-atree-register-inlining.22]: https://github.com/onflow/cadence/compare/v1.0.0-preview.21...v1.0.0-preview-atree-register-inlining.22
[v1.0.0-preview.21]: https://github.com/onflow/cadence/compare/v1.0.0-preview.20...v1.0.0-preview.21
[v1.0.0-preview.20]: https://github.com/onflow/cadence/compare/v0.42.10...v1.0.0-preview.20
[v0.42.10]: https://github.com/onflow/cadence/compare/v1.0.0-preview.19...v0.42.10
[v1.0.0-preview.19]: https://github.com/onflow/cadence/compare/v1.0.0-preview.18...v1.0.0-preview.19
[v1.0.0-preview.18]: https://github.com/onflow/cadence/compare/v1.0.0-preview.17...v1.0.0-preview.18
[v1.0.0-preview.17]: https://github.com/onflow/cadence/compare/v1.0.0-preview.16...v1.0.0-preview.17
[v1.0.0-preview.16]: https://github.com/onflow/cadence/compare/v1.0.0-preview.15...v1.0.0-preview.16
[v1.0.0-preview.15]: https://github.com/onflow/cadence/compare/v1.0.0-preview.14...v1.0.0-preview.15
[v1.0.0-preview.14]: https://github.com/onflow/cadence/compare/v1.0.0-preview.13...v1.0.0-preview.14
[v1.0.0-preview.13]: https://github.com/onflow/cadence/compare/v1.0.0-preview.12...v1.0.0-preview.13
[v1.0.0-preview.12]: https://github.com/onflow/cadence/compare/v1.0.0-preview.11...v1.0.0-preview.12
[v1.0.0-preview.11]: https://github.com/onflow/cadence/compare/v1.0.0-preview.10...v1.0.0-preview.11
[v1.0.0-preview.10]: https://github.com/onflow/cadence/compare/v1.0.0-M9...v1.0.0-preview.10
[v1.0.0-M9]: https://github.com/onflow/cadence/compare/v1.0.0-M8...v1.0.0-M9
[v1.0.0-M8]: https://github.com/onflow/cadence/compare/v1.0.0-M7...v1.0.0-M8
[v1.0.0-M7]: https://github.com/onflow/cadence/compare/v1.0.0-M6...v1.0.0-M7
[v1.0.0-M6]: https://github.com/onflow/cadence/compare/v0.42.9...v1.0.0-M6
[v0.42.9]: https://github.com/onflow/cadence/compare/v1.0.0-M5...v0.42.9
[v1.0.0-M5]: https://github.com/onflow/cadence/compare/v1.0.0-M4...v1.0.0-M5
[v1.0.0-M4]: https://github.com/onflow/cadence/compare/v0.42.8...v1.0.0-M4
[v0.42.8]: https://github.com/onflow/cadence/compare/v1.0.0-M3...v0.42.8
[v1.0.0-M3]: https://github.com/onflow/cadence/compare/v1.0.0-preview.2...v1.0.0-M3
[v1.0.0-preview.2]: https://github.com/onflow/cadence/compare/v0.42.7...v1.0.0-preview.2
[v0.42.7]: https://github.com/onflow/cadence/compare/v0.42.6...v0.42.7
[v0.42.6]: https://github.com/onflow/cadence/compare/v0.42.5...v0.42.6
[v0.42.5]: https://github.com/onflow/cadence/compare/v0.42.4...v0.42.5
[v0.42.4]: https://github.com/onflow/cadence/compare/v0.42.3...v0.42.4
[v0.42.3]: https://github.com/onflow/cadence/compare/v0.42.2...v0.42.3
[v0.42.2]: https://github.com/onflow/cadence/compare/v0.39.17...v0.42.2
[v0.39.17]: https://github.com/onflow/cadence/compare/v0.42.1...v0.39.17
[v0.42.1]: https://github.com/onflow/cadence/compare/v0.42.0...v0.42.1
[v0.42.0]: https://github.com/onflow/cadence/compare/v0.39.16...v0.42.0
[v0.39.16]: https://github.com/onflow/cadence/compare/v1.0.0-preview.1...v0.39.16
[v1.0.0-preview.1]: https://github.com/onflow/cadence/compare/v0.41.1...v1.0.0-preview.1
[v0.41.1]: https://github.com/onflow/cadence/compare/v0.39.15...v0.41.1
[v0.39.15]: https://github.com/onflow/cadence/compare/v0.41.0...v0.39.15
[v0.41.0]: https://github.com/onflow/cadence/compare/v0.41.0-stable-cadence.1...v0.41.0
[v0.41.0-stable-cadence.1]: https://github.com/onflow/cadence/compare/v0.40.0...v0.41.0-stable-cadence.1
[v0.40.0]: https://github.com/onflow/cadence/compare/v0.39.14...v0.40.0
[v0.39.14]: https://github.com/onflow/cadence/compare/v0.39.13-stable-cadence...v0.39.14
[v0.39.13-stable-cadence]: https://github.com/onflow/cadence/compare/v0.39.12...v0.39.13-stable-cadence
[v0.39.12]: https://github.com/onflow/cadence/compare/v0.39.11...v0.39.12
[v0.39.11]: https://github.com/onflow/cadence/compare/v0.39.10...v0.39.11
[v0.39.10]: https://github.com/onflow/cadence/compare/v0.39.9...v0.39.10
[v0.39.9]: https://github.com/onflow/cadence/compare/v0.39.8...v0.39.9
[v0.39.8]: https://github.com/onflow/cadence/compare/v0.39.7...v0.39.8
[v0.39.7]: https://github.com/onflow/cadence/compare/v0.39.6...v0.39.7
[v0.39.6]: https://github.com/onflow/cadence/compare/v0.39.5...v0.39.6
[v0.39.5]: https://github.com/onflow/cadence/compare/v0.39.4...v0.39.5
[v0.39.4]: https://github.com/onflow/cadence/compare/v0.39.3...v0.39.4
[v0.39.3]: https://github.com/onflow/cadence/compare/v0.39.2...v0.39.3
[v0.39.2]: https://github.com/onflow/cadence/compare/v0.31.6...v0.39.2
[v0.31.6]: https://github.com/onflow/cadence/compare/v0.39.1...v0.31.6
[v0.39.1]: https://github.com/onflow/cadence/compare/v0.39.0...v0.39.1
[v0.39.0]: https://github.com/onflow/cadence/compare/v0.38.1...v0.39.0
[v0.38.1]: https://github.com/onflow/cadence/compare/v0.38.0...v0.38.1
[v0.38.0]: https://github.com/onflow/cadence/compare/v0.37.0...v0.38.0
[v0.37.0]: https://github.com/onflow/cadence/compare/v0.36.0...v0.37.0
[v0.36.0]: https://github.com/onflow/cadence/compare/v0.35.0...v0.36.0
[v0.35.0]: https://github.com/onflow/cadence/compare/v0.31.5...v0.35.0
[v0.31.5]: https://github.com/onflow/cadence/compare/v0.34.0...v0.31.5
[v0.34.0]: https://github.com/onflow/cadence/compare/v0.33.0...v0.34.0
[v0.33.0]: https://github.com/onflow/cadence/compare/v0.31.3...v0.33.0
[v0.31.3]: https://github.com/onflow/cadence/compare/v0.31.2...v0.31.3
[v0.31.2]: https://github.com/onflow/cadence/compare/v0.31.1...v0.31.2
[v0.31.1]: https://github.com/onflow/cadence/compare/v0.31.0...v0.31.1
[v0.31.0]: https://github.com/onflow/cadence/compare/v0.30.0...v0.31.0
[v0.30.0]: https://github.com/onflow/cadence/compare/v0.29.0...v0.30.0
[v0.29.0]: https://github.com/onflow/cadence/compare/v0.29.0-stable-cadence...v0.29.0
[v0.29.0-stable-cadence]: https://github.com/onflow/cadence/compare/v0.28.0...v0.29.0-stable-cadence
[v0.28.0]: https://github.com/onflow/cadence/compare/v0.27.1...v0.28.0
[v0.27.1]: https://github.com/onflow/cadence/compare/v0.27.0...v0.27.1
[v0.27.0]: https://github.com/onflow/cadence/compare/v0.26.2...v0.27.0
[v0.26.2]: https://github.com/onflow/cadence/compare/v0.25.2...v0.26.2
[v0.25.2]: https://github.com/onflow/cadence/compare/v0.25.1...v0.25.2
[v0.25.1]: https://github.com/onflow/cadence/compare/v0.26.1...v0.25.1
[v0.26.1]: https://github.com/onflow/cadence/compare/languageserver/v0.26.2...v0.26.1
[languageserver/v0.26.2]: https://github.com/onflow/cadence/compare/languageserver/v0.26.1...languageserver/v0.26.2
[languageserver/v0.26.1]: https://github.com/onflow/cadence/compare/v0.26.0...languageserver/v0.26.1
[v0.26.0]: https://github.com/onflow/cadence/compare/languageserver/v0.26.0...v0.26.0
[languageserver/v0.26.0]: https://github.com/onflow/cadence/compare/v0.25.0...languageserver/v0.26.0
[v0.25.0]: https://github.com/onflow/cadence/compare/languageserver/v0.25.0...v0.25.0
[languageserver/v0.25.0]: https://github.com/onflow/cadence/compare/v0.24.6...languageserver/v0.25.0
[v0.24.6]: https://github.com/onflow/cadence/compare/v0.24.5...v0.24.6
[v0.24.5]: https://github.com/onflow/cadence/compare/v0.24.4...v0.24.5
[v0.24.4]: https://github.com/onflow/cadence/compare/v0.24.3...v0.24.4
[v0.24.3]: https://github.com/onflow/cadence/compare/v0.24.2...v0.24.3
[v0.24.2]: https://github.com/onflow/cadence/compare/v0.24.1...v0.24.2
[v0.24.1]: https://github.com/onflow/cadence/compare/languageserver/v0.24.0...v0.24.1
[languageserver/v0.24.0]: https://github.com/onflow/cadence/compare/v0.24.0...languageserver/v0.24.0
[v0.24.0]: https://github.com/onflow/cadence/compare/languageserver/v0.23.4...v0.24.0
[languageserver/v0.23.4]: https://github.com/onflow/cadence/compare/languageserver/v0.23.0...languageserver/v0.23.4
[languageserver/v0.23.0]: https://github.com/onflow/cadence/compare/v0.23.4...languageserver/v0.23.0
[v0.23.4]: https://github.com/onflow/cadence/compare/v0.23.3...v0.23.4
[v0.23.3]: https://github.com/onflow/cadence/compare/v0.23.2...v0.23.3
[v0.23.2]: https://github.com/onflow/cadence/compare/v0.23.1...v0.23.2
[v0.23.1]: https://github.com/onflow/cadence/compare/v0.23.0...v0.23.1
[v0.23.0]: https://github.com/onflow/cadence/compare/v0.21.3...v0.23.0
[v0.21.3]: https://github.com/onflow/cadence/compare/v0.21.2...v0.21.3
[v0.21.2]: https://github.com/onflow/cadence/compare/v0.21.1...v0.21.2
[v0.21.1]: https://github.com/onflow/cadence/compare/v0.21.0...v0.21.1
[v0.21.0]: https://github.com/onflow/cadence/compare/v0.20.0...v0.21.0
[v0.20.0]: https://github.com/onflow/cadence/compare/v0.19.1...v0.20.0
[v0.19.1]: https://github.com/onflow/cadence/compare/v0.19.0...v0.19.1
[v0.19.0]: https://github.com/onflow/cadence/compare/v0.18.0...v0.19.0
[v0.18.0]: https://github.com/onflow/cadence/compare/v0.17.0...v0.18.0
[v0.17.0]: https://github.com/onflow/cadence/compare/v0.16.1...v0.17.0
[v0.16.1]: https://github.com/onflow/cadence/compare/v0.16.0...v0.16.1
[v0.16.0]: https://github.com/onflow/cadence/compare/v0.15.1...v0.16.0
[v0.15.1]: https://github.com/onflow/cadence/compare/v0.15.0...v0.15.1
[v0.15.0]: https://github.com/onflow/cadence/compare/v0.14.5...v0.15.0
[v0.14.5]: https://github.com/onflow/cadence/compare/v0.14.4...v0.14.5
[v0.14.4]: https://github.com/onflow/cadence/compare/v0.13.10...v0.14.4
[v0.13.10]: https://github.com/onflow/cadence/compare/v0.13.9...v0.13.10
[v0.13.9]: https://github.com/onflow/cadence/compare/v0.14.3...v0.13.9
[v0.14.3]: https://github.com/onflow/cadence/compare/v0.14.2...v0.14.3
[v0.14.2]: https://github.com/onflow/cadence/compare/v0.13.8...v0.14.2
[v0.13.8]: https://github.com/onflow/cadence/compare/v0.14.1...v0.13.8
[v0.14.1]: https://github.com/onflow/cadence/compare/v0.14.0...v0.14.1
[v0.14.0]: https://github.com/onflow/cadence/compare/v0.13.6...v0.14.0
[v0.13.6]: https://github.com/onflow/cadence/compare/v0.13.5...v0.13.6
[v0.13.5]: https://github.com/onflow/cadence/compare/v0.13.4...v0.13.5
[v0.13.4]: https://github.com/onflow/cadence/compare/v0.13.3...v0.13.4
[v0.13.3]: https://github.com/onflow/cadence/compare/v0.13.2...v0.13.3
[v0.13.2]: https://github.com/onflow/cadence/compare/v0.13.1...v0.13.2
[v0.13.1]: https://github.com/onflow/cadence/compare/v0.12.12...v0.13.1
[v0.12.12]: https://github.com/onflow/cadence/compare/v0.13.0...v0.12.12
[v0.13.0]: https://github.com/onflow/cadence/compare/v0.12.11...v0.13.0
[v0.12.11]: https://github.com/onflow/cadence/compare/v0.12.10...v0.12.11
[v0.12.10]: https://github.com/onflow/cadence/compare/v0.12.9...v0.12.10
[v0.12.9]: https://github.com/onflow/cadence/compare/v0.12.8...v0.12.9
[v0.12.8]: https://github.com/onflow/cadence/compare/v0.12.7...v0.12.8
[v0.12.7]: https://github.com/onflow/cadence/compare/v0.12.6...v0.12.7
[v0.12.6]: https://github.com/onflow/cadence/compare/v0.10.6...v0.12.6
[v0.10.6]: https://github.com/onflow/cadence/compare/v0.12.5...v0.10.6
[v0.12.5]: https://github.com/onflow/cadence/compare/v0.12.4...v0.12.5
[v0.12.4]: https://github.com/onflow/cadence/compare/v0.12.3...v0.12.4
[v0.12.3]: https://github.com/onflow/cadence/compare/v0.12.2...v0.12.3
[v0.12.2]: https://github.com/onflow/cadence/compare/v0.12.1...v0.12.2
[v0.12.1]: https://github.com/onflow/cadence/compare/v0.10.5...v0.12.1
[v0.10.5]: https://github.com/onflow/cadence/compare/v0.12.0...v0.10.5
[v0.12.0]: https://github.com/onflow/cadence/compare/v0.10.4...v0.12.0
[v0.10.4]: https://github.com/onflow/cadence/compare/v0.10.3...v0.10.4
[v0.10.3]: https://github.com/onflow/cadence/compare/v0.11.2...v0.10.3
[v0.11.2]: https://github.com/onflow/cadence/compare/v0.10.2...v0.11.2
[v0.10.2]: https://github.com/onflow/cadence/compare/v0.9.3...v0.10.2
[v0.9.3]: https://github.com/onflow/cadence/compare/v0.11.1...v0.9.3
[v0.11.1]: https://github.com/onflow/cadence/compare/v0.10.1...v0.11.1
[v0.10.1]: https://github.com/onflow/cadence/compare/v0.11.0...v0.10.1
[v0.11.0]: https://github.com/onflow/cadence/compare/v0.9.2...v0.11.0
[v0.9.2]: https://github.com/onflow/cadence/compare/v0.10.0...v0.9.2
[v0.10.0]: https://github.com/onflow/cadence/compare/v0.9.0...v0.10.0
[v0.9.0]: https://github.com/onflow/cadence/compare/v0.8.2...v0.9.0
[v0.8.2]: https://github.com/onflow/cadence/compare/v0.8.1...v0.8.2
[v0.8.1]: https://github.com/onflow/cadence/compare/v0.8.0...v0.8.1
[v0.8.0]: https://github.com/onflow/cadence/compare/v0.7.0...v0.8.0
[v0.7.0]: https://github.com/onflow/cadence/compare/v0.6.0...v0.7.0
[v0.6.0]: https://github.com/onflow/cadence/compare/v0.5.1...v0.6.0
[v0.5.1]: https://github.com/onflow/cadence/compare/v0.5.0...v0.5.1
[v0.5.0]: https://github.com/onflow/cadence/compare/v0.4.0...v0.5.0
[v0.4.0]: https://github.com/onflow/cadence/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/onflow/cadence/tree/v0.3.0

<!-- Generated by https://github.com/rhysd/changelog-from-release v3.7.2 -->
