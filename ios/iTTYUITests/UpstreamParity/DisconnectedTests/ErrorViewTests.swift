//
//  ErrorViewTests.swift
//  iTTYUITests
//
//  Verifies the simulator fallback for manual setup flows. Because live
//  terminal sessions are not supported in the iOS simulator, manual SSH
//  paths should stay stable and show an explanatory error instead of
//  crashing while mounting Ghostty.
//

import XCTest

final class ErrorViewTests: XCTestCase {

    var app: XCUIApplication!

    override func setUpWithError() throws {
        continueAfterFailure = false
        app = launchForDisconnectedTests()

        let manualSetup = app.buttons["DisconnectedSavedConnectionsButton"]
        XCTAssertTrue(manualSetup.waitForExistence(timeout: 5))
        manualSetup.tap()

        let navBar = app.navigationBars["Connections"]
        XCTAssertTrue(navBar.waitForExistence(timeout: 3))

        let quickConnect = app.buttons["QuickConnectButton"]
        XCTAssertTrue(quickConnect.waitForExistence(timeout: 3))
        quickConnect.tap()

        let hostField = app.textFields["HostField"]
        XCTAssertTrue(hostField.waitForExistence(timeout: 3))
    }

    override func tearDownWithError() throws {
        takeScreenshot(app, name: "\(name)-teardown")
        app = nil
    }

    func testSimulatorQuickConnectShowsInlineMessage() throws {
        let hostField = app.textFields["HostField"]
        hostField.tap()
        hostField.typeText("example.tailnet.ts.net")

        let usernameField = app.textFields["UsernameField"]
        usernameField.tap()
        usernameField.typeText("jacob")

        let connectButton = app.buttons["ConnectButton"]
        XCTAssertTrue(connectButton.isEnabled)
        connectButton.tap()

        let errorMessage = app.staticTexts["QuickConnectErrorMessage"]
        XCTAssertTrue(errorMessage.waitForExistence(timeout: 3))
        XCTAssertEqual(
            errorMessage.label,
            "Terminal sessions are not available in the iOS simulator yet. Use a physical device to open a live terminal."
        )

        takeScreenshot(app, name: "Error-01-SimulatorInlineMessage")
    }

    func testSimulatorQuickConnectCanCancelAfterError() throws {
        let hostField = app.textFields["HostField"]
        hostField.tap()
        hostField.typeText("example.tailnet.ts.net")

        let usernameField = app.textFields["UsernameField"]
        usernameField.tap()
        usernameField.typeText("jacob")

        app.buttons["ConnectButton"].tap()
        XCTAssertTrue(app.staticTexts["QuickConnectErrorMessage"].waitForExistence(timeout: 3))

        app.buttons["QuickConnectCancelButton"].tap()

        let navBar = app.navigationBars["Connections"]
        XCTAssertTrue(navBar.waitForExistence(timeout: 3))
        XCTAssertFalse(app.isOnErrorScreen)

        takeScreenshot(app, name: "Error-02-BackToConnections")
    }
}
