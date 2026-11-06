#!/usr/bin/env bash
set -euo pipefail

APP_NAME="MacHole"
BUNDLE_DIR="${APP_NAME}.app"
ICONSET_DIR="AppIcon.iconset"
UNIVERSAL=1
BUILD_DMG=0

for arg in "$@"; do
  case "$arg" in
    --native)
      UNIVERSAL=0
      ;;
    --dmg)
      BUILD_DMG=1
      ;;
  esac
done

echo "🎨 Generating native macOS AppIcon.icns..."
rm -rf "${ICONSET_DIR}" AppIcon.icns
mkdir -p "${ICONSET_DIR}"

# Render high-resolution master icon using native macOS AppKit via swift
swift - << 'SWIFT'
import AppKit

let size = CGSize(width: 1024, height: 1024)
let image = NSImage(size: size)
image.lockFocus()

guard let ctx = NSGraphicsContext.current?.cgContext else { exit(1) }

// macOS squircle background
let rect = CGRect(x: 82, y: 82, width: 860, height: 860)
let path = NSBezierPath(roundedRect: rect, xRadius: 195, yRadius: 195)

let bgGradient = NSGradient(starting: NSColor(srgbRed: 0.17, green: 0.17, blue: 0.19, alpha: 1.0),
                            ending: NSColor(srgbRed: 0.10, green: 0.10, blue: 0.12, alpha: 1.0))!
bgGradient.draw(in: path, angle: -90)

// Subtle inner border
NSColor(srgbRed: 1.0, green: 1.0, blue: 1.0, alpha: 0.16).setStroke()
path.lineWidth = 14
path.stroke()

// Vector symbol
ctx.setStrokeColor(NSColor(srgbRed: 0.95, green: 0.95, blue: 0.97, alpha: 1.0).cgColor)
ctx.setLineWidth(48)
ctx.setLineCap(.round)
ctx.setLineJoin(.round)

let center = CGPoint(x: 512, y: 512)
ctx.strokeEllipse(in: CGRect(x: center.x - 252, y: center.y - 252, width: 504, height: 504))
ctx.strokeEllipse(in: CGRect(x: center.x - 84, y: center.y - 84, width: 168, height: 168))

ctx.move(to: CGPoint(x: center.x, y: center.y + 84))
ctx.addLine(to: CGPoint(x: center.x, y: center.y + 252))
ctx.strokePath()

ctx.move(to: CGPoint(x: center.x, y: center.y - 84))
ctx.addLine(to: CGPoint(x: center.x, y: center.y - 252))
ctx.strokePath()

image.unlockFocus()

if let tiff = image.tiffRepresentation,
   let rep = NSBitmapImageRep(data: tiff),
   let png = rep.representation(using: .png, properties: [:]) {
    try! png.write(to: URL(fileURLWithPath: "AppIcon.iconset/icon_512x512@2x.png"))
}
SWIFT

MASTER="${ICONSET_DIR}/icon_512x512@2x.png"
sips -z 16 16     "${MASTER}" --out "${ICONSET_DIR}/icon_16x16.png"      >/dev/null
sips -z 32 32     "${MASTER}" --out "${ICONSET_DIR}/icon_16x16@2x.png"   >/dev/null
sips -z 32 32     "${MASTER}" --out "${ICONSET_DIR}/icon_32x32.png"      >/dev/null
sips -z 64 64     "${MASTER}" --out "${ICONSET_DIR}/icon_32x32@2x.png"   >/dev/null
sips -z 128 128   "${MASTER}" --out "${ICONSET_DIR}/icon_128x128.png"    >/dev/null
sips -z 256 256   "${MASTER}" --out "${ICONSET_DIR}/icon_128x128@2x.png" >/dev/null
sips -z 256 256   "${MASTER}" --out "${ICONSET_DIR}/icon_256x256.png"    >/dev/null
sips -z 512 512   "${MASTER}" --out "${ICONSET_DIR}/icon_256x256@2x.png" >/dev/null
sips -z 512 512   "${MASTER}" --out "${ICONSET_DIR}/icon_512x512.png"    >/dev/null

iconutil -c icns "${ICONSET_DIR}" -o AppIcon.icns
rm -rf "${ICONSET_DIR}"

if [ "${UNIVERSAL}" -eq 1 ]; then
  echo "🔨 Compiling Universal Binary (arm64 + amd64)..."
  CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o "${APP_NAME}_arm64" ./cmd/machole
  CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o "${APP_NAME}_amd64" ./cmd/machole
  lipo -create -output "${APP_NAME}" "${APP_NAME}_arm64" "${APP_NAME}_amd64"
  rm -f "${APP_NAME}_arm64" "${APP_NAME}_amd64"
else
  echo "🔨 Compiling native architecture binary..."
  CGO_ENABLED=1 go build -ldflags="-s -w" -o "${APP_NAME}" ./cmd/machole
fi

echo "📦 Packaging macOS application bundle (${BUNDLE_DIR})..."
rm -rf "${BUNDLE_DIR}"
mkdir -p "${BUNDLE_DIR}/Contents/MacOS"
mkdir -p "${BUNDLE_DIR}/Contents/Resources"

cp "${APP_NAME}" "${BUNDLE_DIR}/Contents/MacOS/${APP_NAME}"
chmod +x "${BUNDLE_DIR}/Contents/MacOS/${APP_NAME}"
mv AppIcon.icns "${BUNDLE_DIR}/Contents/Resources/AppIcon.icns"

cat > "${BUNDLE_DIR}/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>${APP_NAME}</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>CFBundleIdentifier</key>
    <string>io.github.0vj.machole</string>
    <key>CFBundleName</key>
    <string>${APP_NAME}</string>
    <key>CFBundleDisplayName</key>
    <string>${APP_NAME}</string>
    <key>CFBundleVersion</key>
    <string>1.1.0</string>
    <key>CFBundleShortVersionString</key>
    <string>1.1.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>LSMinimumSystemVersion</key>
    <string>13.0</string>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOF

echo "🔏 Performing ad-hoc code signature..."
codesign --force --deep -s - "${BUNDLE_DIR}"
touch "${BUNDLE_DIR}"

if [ "${BUILD_DMG}" -eq 1 ]; then
  DMG_NAME="${APP_NAME}-macOS-Universal.dmg"
  echo "💿 Creating DMG installer (${DMG_NAME})..."
  STAGING="dmg_staging"
  rm -rf "${STAGING}" "${DMG_NAME}"
  mkdir -p "${STAGING}"
  cp -R "${BUNDLE_DIR}" "${STAGING}/"
  ln -s /Applications "${STAGING}/Applications"
  hdiutil create -volname "${APP_NAME}" -srcfolder "${STAGING}" -ov -format UDZO "${DMG_NAME}"
  rm -rf "${STAGING}"
  echo "✅ DMG created: ./${DMG_NAME}"
fi

echo "✅ Build complete!"
echo "▶️ Universal binary: ./${APP_NAME}"
echo "▶️ Application bundle: ./${BUNDLE_DIR}"
