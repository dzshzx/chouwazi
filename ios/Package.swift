// swift-tools-version: 5.9
import PackageDescription
let package = Package(name: "DrawCore", products: [.library(name: "DrawCore", targets: ["DrawCore"])], targets: [.target(name: "DrawCore"), .testTarget(name: "DrawCoreTests", dependencies: ["DrawCore"])])
