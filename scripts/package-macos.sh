#!/usr/bin/env bash
set -euo pipefail

version=${1:?Usage: package-macos.sh VERSION}
export GOWORK=off CGO_ENABLED=1
mkdir -p dist build/cli
for arch in amd64 arm64; do
  if [[ "$arch" == amd64 ]]; then cc='clang -arch x86_64'; else cc='clang -arch arm64'; fi
  (cd backend && GOARCH="$arch" CC="$cc" go build -trimpath -ldflags "-s -w -X backend-go/app.Version=$version" -o "../build/cli/skm-$arch" ./cmd/skm)
done
app=build/bin/SKM.app
test -d "$app"
mkdir -p "$app/Contents/Resources"
lipo -create build/cli/skm-amd64 build/cli/skm-arm64 -output "$app/Contents/Resources/skm"
lipo "$app/Contents/MacOS/SKM" -verify_arch x86_64 arm64
lipo "$app/Contents/Resources/skm" -verify_arch x86_64 arm64
short_version=${version#v}
short_version=${short_version%%-*}
if [[ ! "$short_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then short_version=0.0.0; fi
/usr/libexec/PlistBuddy -c "Set :CFBundleVersion $short_version" "$app/Contents/Info.plist"
/usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $short_version" "$app/Contents/Info.plist"
# Adding the CLI changes the bundle; refresh the ad-hoc signature afterwards.
codesign --force --deep --sign - "$app"
codesign --verify --deep --strict "$app"
actual=$("$app/Contents/Resources/skm" version)
test "$actual" = "$version"
ditto -c -k --sequesterRsrc --keepParent "$app" "dist/SKM-$version-macos-universal.zip"
