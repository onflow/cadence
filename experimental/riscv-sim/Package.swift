// swift-tools-version:6.0
// The swift-tools-version declares the minimum version of Swift required to build this package.

import PackageDescription

let package = Package(
    name: "riscv-sim",
    platforms: [
        .macOS(.v15),
    ],
    products: [
        .executable( name: "riscv-sim", targets: [ "riscv-sim" ] ),
    ],
    targets: [
        // Targets are the basic building blocks of a package. A target can define a module or a test suite.
        // Targets can depend on other targets in this package, and on products in packages which this package depends on.
        .executableTarget(
            name: "riscv-sim",
            path: "Sources/riscv-sim" ),
    ]
)
