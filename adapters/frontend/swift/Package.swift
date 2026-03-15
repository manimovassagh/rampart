// swift-tools-version: 5.9

import PackageDescription

let package = Package(
    name: "Rampart",
    platforms: [
        .iOS(.v15),
        .macOS(.v12),
    ],
    products: [
        .library(
            name: "Rampart",
            targets: ["Rampart"]
        ),
    ],
    targets: [
        .target(
            name: "Rampart",
            path: "Sources/Rampart"
        ),
    ]
)
