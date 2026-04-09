# This formula is updated automatically by GoReleaser on each release.
# Manual install: brew install Rohit-554/tap/droidlink
#
# To add the tap: brew tap Rohit-554/tap
class Droidlink < Formula
  desc "Robust WiFi ADB CLI for Android developers — auto-reconnects, mDNS discovery, install queue"
  homepage "https://github.com/Rohit-554/droidLink"
  version "0.0.0" # placeholder — GoReleaser fills this in

  on_macos do
    on_arm do
      url "https://github.com/Rohit-554/droidLink/releases/download/v#{version}/droidlink_darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER_ARM64_SHA256"
    end
    on_intel do
      url "https://github.com/Rohit-554/droidLink/releases/download/v#{version}/droidlink_darwin_amd64.tar.gz"
      sha256 "PLACEHOLDER_AMD64_SHA256"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Rohit-554/droidLink/releases/download/v#{version}/droidlink_linux_arm64.tar.gz"
      sha256 "PLACEHOLDER_LINUX_ARM64_SHA256"
    end
    on_intel do
      url "https://github.com/Rohit-554/droidLink/releases/download/v#{version}/droidlink_linux_amd64.tar.gz"
      sha256 "PLACEHOLDER_LINUX_AMD64_SHA256"
    end
  end

  def install
    bin.install "droidlink"
  end

  test do
    system "#{bin}/droidlink", "--help"
  end
end
