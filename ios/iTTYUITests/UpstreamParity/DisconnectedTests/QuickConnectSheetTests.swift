//
//  QuickConnectSheetTests.swift
//  iTTYUITests
//
//  Covers the disconnected-screen primary flow after the Tailscale-first
//  redesign. The home action now opens discovery instead of the legacy
//  direct-SSH quick connect sheet.
//

import XCTest

final class TailscaleDiscoveryTests: XCTestCase {

    var app: XCUIApplication!

    override func setUpWithError() throws {
        continueAfterFailure = false
        app = launchForDisconnectedTests()

        let findComputers = app.buttons["DisconnectedQuickConnectButton"]
        XCTAssertTrue(findComputers.waitForExistence(timeout: 5))
        findComputers.tap()

        let navBar = app.navigationBars["Find Computers"]
        XCTAssertTrue(navBar.waitForExistence(timeout: 3),
                      "Discovery view should appear from the disconnected screen")
    }

    override func tearDownWithError() throws {
        takeScreenshot(app, name: "\(name)-teardown")
        app = nil
    }

    func testDiscoveryScreenShowsPrimaryActions() throws {
        XCTAssertTrue(app.staticTexts["TailscaleDiscoveryTitle"].exists)
        XCTAssertTrue(app.buttons["TailscaleRefreshButton"].exists)
        XCTAssertTrue(app.buttons["TailscaleAddComputerButton"].exists)
        XCTAssertTrue(app.buttons["AddComputerManuallyButton"].exists)
        XCTAssertTrue(app.buttons["ManualSSHSetupButton"].exists)

        takeScreenshot(app, name: "Discovery-01-PrimaryActions")
    }

    func testManualSSHSetupOpensConnectionList() throws {
        let manualSetup = app.buttons["ManualSSHSetupButton"]
        XCTAssertTrue(manualSetup.exists)
        manualSetup.tap()

        let navBar = app.navigationBars["Connections"]
        XCTAssertTrue(navBar.waitForExistence(timeout: 3))
        XCTAssertTrue(app.buttons["QuickConnectButton"].exists)

        takeScreenshot(app, name: "Discovery-02-ManualSetup")
    }

    func testAddComputerOpensComputerEditor() throws {
        let addComputer = app.buttons["TailscaleAddComputerButton"]
        XCTAssertTrue(addComputer.exists)
        addComputer.tap()

        let navBar = app.navigationBars["Add Computer"]
        XCTAssertTrue(navBar.waitForExistence(timeout: 3))
        XCTAssertTrue(app.textFields["Name"].exists)
        XCTAssertTrue(app.textFields["Daemon Hostname"].exists)

        takeScreenshot(app, name: "Discovery-03-AddComputer")
    }

    func testDoneDismissesDiscovery() throws {
        app.buttons["Done"].tap()

        XCTAssertTrue(app.waitForDisconnectedScreen(timeout: 3),
                      "Done should return to the disconnected screen")

        takeScreenshot(app, name: "Discovery-04-Dismissed")
    }
}
