#!/bin/bash
# Migrate content from book/tmp to canonical publish/content location
# This script handles the chapter remapping from the old structure

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PUBLISH_DIR="$(dirname "$SCRIPT_DIR")"
ROOT_DIR="$(dirname "$PUBLISH_DIR")"
SOURCE_DIR="$ROOT_DIR/book/tmp"
CONTENT_DIR="$PUBLISH_DIR/content"

echo "=========================================="
echo "Migrating Book Content"
echo "=========================================="
echo ""
echo "Source: $SOURCE_DIR"
echo "Target: $CONTENT_DIR"
echo ""

# Verify source directory exists
if [ ! -d "$SOURCE_DIR" ]; then
    echo "ERROR: Source directory not found: $SOURCE_DIR"
    exit 1
fi

# Function to copy and transform a file
copy_content() {
    local src="$1"
    local dest="$2"
    local filename=$(basename "$dest")

    if [ -f "$src" ]; then
        cp "$src" "$dest"
        echo "  Migrated: $filename"
    else
        echo "  WARNING: Source not found: $src"
    fi
}

# ============================================
# Introduction
# ============================================
echo "Migrating Introduction..."
copy_content "$SOURCE_DIR/ch00_introduction.md" "$CONTENT_DIR/introduction.md"

# ============================================
# Part 1: Getting Started
# ============================================
echo ""
echo "Migrating Part 1: Getting Started..."

# Chapter 1: Why Gorai
CH01_DIR="$CONTENT_DIR/part1-getting-started/ch01-why-gorai"
copy_content "$SOURCE_DIR/ch01_s1_landscape.md" "$CH01_DIR/landscape.md"
copy_content "$SOURCE_DIR/ch01_s2_philosophy.md" "$CH01_DIR/philosophy.md"
copy_content "$SOURCE_DIR/ch01_s3_audience.md" "$CH01_DIR/audience.md"
copy_content "$SOURCE_DIR/ch01_s4_whatyoullbuild.md" "$CH01_DIR/whatyoullbuild.md"
copy_content "$SOURCE_DIR/ch01_s5_prerequisites.md" "$CH01_DIR/prerequisites.md"

# Chapter 2: Architecture
CH02_DIR="$CONTENT_DIR/part1-getting-started/ch02-architecture"
copy_content "$SOURCE_DIR/ch02_s1_bigpicture.md" "$CH02_DIR/bigpicture.md"
copy_content "$SOURCE_DIR/ch02_s2_coreconcepts.md" "$CH02_DIR/coreconcepts.md"
copy_content "$SOURCE_DIR/ch02_s3_distributed.md" "$CH02_DIR/distributed.md"
copy_content "$SOURCE_DIR/ch02_s4_config.md" "$CH02_DIR/config.md"
copy_content "$SOURCE_DIR/ch02_s5_nwsnwc.md" "$CH02_DIR/nwsnwc.md"

# ============================================
# Part 2: Core Framework
# ============================================
echo ""
echo "Migrating Part 2: Core Framework..."

# Chapter 3: NATS Messaging
CH03_DIR="$CONTENT_DIR/part2-core-framework/ch03-nats"
copy_content "$SOURCE_DIR/ch03_s1_whynats.md" "$CH03_DIR/whynats.md"
copy_content "$SOURCE_DIR/ch03_s2_fundamentals.md" "$CH03_DIR/fundamentals.md"
copy_content "$SOURCE_DIR/ch03_s3_patterns.md" "$CH03_DIR/patterns.md"
copy_content "$SOURCE_DIR/ch03_s4_qos.md" "$CH03_DIR/qos.md"
copy_content "$SOURCE_DIR/ch03_s5_jetstream.md" "$CH03_DIR/jetstream.md"
copy_content "$SOURCE_DIR/ch03_s6_cli.md" "$CH03_DIR/cli.md"

# Chapter 4: Sensors
CH04_DIR="$CONTENT_DIR/part2-core-framework/ch04-sensors"
copy_content "$SOURCE_DIR/ch04_s1_interface.md" "$CH04_DIR/interface.md"
copy_content "$SOURCE_DIR/ch04_s2_builtin.md" "$CH04_DIR/builtin.md"
copy_content "$SOURCE_DIR/ch04_s3_datatypes.md" "$CH04_DIR/datatypes.md"
copy_content "$SOURCE_DIR/ch04_s4_fakes.md" "$CH04_DIR/fakes.md"

# Chapter 5: Actuators
CH05_DIR="$CONTENT_DIR/part2-core-framework/ch05-actuators"
copy_content "$SOURCE_DIR/ch05_s1_actuator.md" "$CH05_DIR/actuator.md"
copy_content "$SOURCE_DIR/ch05_s2_motor.md" "$CH05_DIR/motor.md"
copy_content "$SOURCE_DIR/ch05_s3_motortypes.md" "$CH05_DIR/motortypes.md"
copy_content "$SOURCE_DIR/ch05_s4_control.md" "$CH05_DIR/control.md"
copy_content "$SOURCE_DIR/ch05_s5_servo.md" "$CH05_DIR/servo.md"
copy_content "$SOURCE_DIR/ch05_s6_base_arm.md" "$CH05_DIR/base_arm.md"

# Chapter 6: Vision
CH06_DIR="$CONTENT_DIR/part2-core-framework/ch06-vision"
copy_content "$SOURCE_DIR/ch06_s1_camera.md" "$CH06_DIR/camera.md"
copy_content "$SOURCE_DIR/ch06_s2_types.md" "$CH06_DIR/types.md"
copy_content "$SOURCE_DIR/ch06_s3_dataflow.md" "$CH06_DIR/dataflow.md"
copy_content "$SOURCE_DIR/ch06_s4_cv.md" "$CH06_DIR/cv.md"

# Chapter 7: Services (single file chapter)
CH07_DIR="$CONTENT_DIR/part2-core-framework/ch07-services"
copy_content "$SOURCE_DIR/ch07_services.md" "$CH07_DIR/_index.md"

# Chapter 8: Behaviors (placeholder - no content yet)
# Chapter 9: Coordinators (placeholder - no content yet)

# ============================================
# Part 3: Development
# ============================================
echo ""
echo "Migrating Part 3: Development..."

# Chapter 10: Development Environment (was ch08)
CH10_DIR="$CONTENT_DIR/part3-development/ch10-devenv"
copy_content "$SOURCE_DIR/ch08_devenv.md" "$CH10_DIR/_index.md"

# Chapter 11: Hello Sensor (was ch09)
CH11_DIR="$CONTENT_DIR/part3-development/ch11-hello-sensor"
copy_content "$SOURCE_DIR/ch09_s1_overview.md" "$CH11_DIR/overview.md"
copy_content "$SOURCE_DIR/ch09_s2_reader.md" "$CH11_DIR/reader.md"
copy_content "$SOURCE_DIR/ch09_s3_sensor.md" "$CH11_DIR/sensor.md"
copy_content "$SOURCE_DIR/ch09_s4_main.md" "$CH11_DIR/main.md"

# Chapter 12: Building Custom Components (was ch10)
CH12_DIR="$CONTENT_DIR/part3-development/ch12-custom"
copy_content "$SOURCE_DIR/ch10_custom.md" "$CH12_DIR/_index.md"

# Chapter 13: Testing (was ch11)
CH13_DIR="$CONTENT_DIR/part3-development/ch13-testing"
copy_content "$SOURCE_DIR/ch11_testing.md" "$CH13_DIR/_index.md"

# ============================================
# Part 4: Advanced
# ============================================
echo ""
echo "Migrating Part 4: Advanced..."

# Chapter 14: AI/ML (was ch12)
CH14_DIR="$CONTENT_DIR/part4-advanced/ch14-ai-ml"
copy_content "$SOURCE_DIR/ch12_ml.md" "$CH14_DIR/_index.md"

# Chapter 15: Project Organization (was ch13)
CH15_DIR="$CONTENT_DIR/part4-advanced/ch15-organization"
copy_content "$SOURCE_DIR/ch13_organization.md" "$CH15_DIR/_index.md"

# Chapter 16: AI-Assisted Development (was ch14)
CH16_DIR="$CONTENT_DIR/part4-advanced/ch16-ai-dev"
copy_content "$SOURCE_DIR/ch14_ai_dev.md" "$CH16_DIR/_index.md"

# Chapter 17: Conclusion (was ch15)
CH17_DIR="$CONTENT_DIR/part4-advanced/ch17-conclusion"
copy_content "$SOURCE_DIR/ch15_conclusion.md" "$CH17_DIR/_index.md"

# ============================================
# Appendices
# ============================================
echo ""
echo "Migrating Appendices..."
# The appendices file needs to be split - for now, copy as placeholder
copy_content "$SOURCE_DIR/appendices.md" "$CONTENT_DIR/appendices/full-appendices.md"

echo ""
echo "=========================================="
echo "Migration Complete!"
echo "=========================================="
echo ""
echo "Content migrated to: $CONTENT_DIR"
echo ""
echo "Note: Some chapters may need additional editing:"
echo "  - Single-file chapters copied to _index.md"
echo "  - Appendices need to be split into individual files"
echo "  - Chapter/section headers may need adjustment"
echo ""
