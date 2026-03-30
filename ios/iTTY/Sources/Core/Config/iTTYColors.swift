// iTTY — Color Palette
//
// Muted steel blue palette for the app chrome (server list, settings, etc.)
// Terminal views use the selected Ghostty theme independently.

import SwiftUI

enum iTTYColors {
    // Primary backgrounds
    static let background = Color(hex: 0x2B3A4A)
    static let surface = Color(hex: 0x354B5E)
    static let surfaceElevated = Color(hex: 0x3E5770)

    // Text
    static let textPrimary = Color.white
    static let textSecondary = Color(hex: 0xB0C4D8)

    // Accent
    static let accent = Color(hex: 0x7EB8E0)

    // Separators and borders
    static let separator = Color(hex: 0x1E2D3A)
    static let border = Color.white.opacity(0.1)

    // Status
    static let online = Color(hex: 0x6BCB77)
    static let offline = Color(hex: 0x8E8E93)
    static let warning = Color(hex: 0xE8A838)
}

extension Color {
    init(hex: UInt, alpha: Double = 1.0) {
        self.init(
            .sRGB,
            red: Double((hex >> 16) & 0xFF) / 255.0,
            green: Double((hex >> 8) & 0xFF) / 255.0,
            blue: Double(hex & 0xFF) / 255.0,
            opacity: alpha
        )
    }
}
