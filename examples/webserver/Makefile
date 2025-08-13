# Makefile for building Lua dynamic libraries
# Modular design to support multiple Lua versions

# =============================================================================
# COMMON PLATFORM CONFIGURATION
# =============================================================================

# Detect OS and architecture
OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH := $(shell uname -m)

# Platform-specific library settings
ifeq ($(OS),linux)
    LIB_PREFIX := lib
    LIB_EXT := so
    LD_FLAGS = -shared -fPIC
    PLATFORM_CFLAGS = -DLUA_USE_LINUX
else ifeq ($(OS),darwin)
    LIB_PREFIX := lib
    LIB_EXT := dylib
    LD_FLAGS = -shared -fPIC
    PLATFORM_CFLAGS = -DLUA_USE_MACOSX
else
    LIB_PREFIX := 
    LIB_EXT := dll
    OS := windows
    LD_FLAGS = -shared
    PLATFORM_CFLAGS = -DLUA_BUILD_AS_DLL
endif

# Common compiler settings
CC = gcc
BASE_CFLAGS = -O2 -Wall -Wextra -fPIC $(PLATFORM_CFLAGS)
LIBS = -lm

# =============================================================================
# COMMON LUA BUILD FUNCTIONS
# =============================================================================

# Common header files for all Lua versions
LUA_COMMON_HEADERS = lua.h luaconf.h lualib.h lauxlib.h

# Common excluded files for all Lua versions
LUA_COMMON_EXCLUDED = lua.c luac.c ltests.c onelua.c

# Function to define Lua version variables
# Usage: $(call define_lua_version,VERSION,MAJOR_MINOR)
define define_lua_version
LUA$(1)_VERSION := $(1)
LUA$(1)_SRC := lua$(2)
LUA$(1)_BUILD_DIR := build/.lua
LUA$(1)_LIB_DIR := $$(LUA$(1)_BUILD_DIR)/lib
LUA$(1)_INCLUDE_DIR := $$(LUA$(1)_BUILD_DIR)/include
LUA$(1)_LIB := $$(LUA$(1)_LIB_DIR)/$$(LIB_PREFIX)lua$(2).$$(LIB_EXT)
LUA$(1)_HEADERS := $$(LUA_COMMON_HEADERS)
LUA$(1)_HEADER_SOURCES := $$(addprefix $$(LUA$(1)_SRC)/, $$(LUA$(1)_HEADERS))
LUA$(1)_HEADER_TARGETS := $$(addprefix $$(LUA$(1)_INCLUDE_DIR)/, $$(LUA$(1)_HEADERS))
LUA$(1)_ALL_SOURCES := $$(wildcard $$(LUA$(1)_SRC)/*.c)
LUA$(1)_EXCLUDED_FILES := $$(addprefix $$(LUA$(1)_SRC)/, $$(LUA_COMMON_EXCLUDED))
LUA$(1)_SOURCES := $$(filter-out $$(LUA$(1)_EXCLUDED_FILES), $$(LUA$(1)_ALL_SOURCES))
LUA$(1)_CORE_OBJS := $$(patsubst %.c,%.o,$$(LUA$(1)_SOURCES))
LUA$(1)_CFLAGS := $$(BASE_CFLAGS)
endef

# =============================================================================
# LUA VERSION DEFINITIONS
# =============================================================================

# Define Lua 5.4
$(eval $(call define_lua_version,54,54))

# =============================================================================
# COMMON BUILD FUNCTIONS
# =============================================================================

# Function to define common build rules for a Lua version
# Usage: $(call define_lua_build_rules,VERSION)
define define_lua_build_rules
# Setup directories for Lua $(1)
.PHONY: setup-dirs-$(1)
setup-dirs-$(1):
	@mkdir -p $$(LUA$(1)_LIB_DIR)
	@mkdir -p $$(LUA$(1)_INCLUDE_DIR)

# Copy header files for Lua $(1)
$$(LUA$(1)_INCLUDE_DIR)/%.h: $$(LUA$(1)_SRC)/%.h
	@cp $$< $$@

# Build Lua $(1) dynamic library
.PHONY: lua$(1)
lua$(1): setup-dirs-$(1) $$(LUA$(1)_LIB) $$(LUA$(1)_HEADER_TARGETS)

# Link Lua $(1) library
$$(LUA$(1)_LIB): $$(LUA$(1)_CORE_OBJS)
	@mkdir -p $$(dir $$@)
	$$(CC) $$(LD_FLAGS) -o $$@ $$^ $$(LIBS)

# Compile Lua $(1) object files
$$(LUA$(1)_SRC)/%.o: $$(LUA$(1)_SRC)/%.c
	$$(CC) $$(LUA$(1)_CFLAGS) -c $$< -o $$@

# Clean Lua $(1) build artifacts
.PHONY: clean-$(1)
clean-$(1):
	rm -f $$(LUA$(1)_SRC)/*.o
	rm -rf $$(LUA$(1)_BUILD_DIR)
endef

# =============================================================================
# VERSION-SPECIFIC BUILD RULES
# =============================================================================

# Generate build rules for Lua 5.4
$(eval $(call define_lua_build_rules,54))

# =============================================================================
# MAIN TARGETS
# =============================================================================

.PHONY: all clean help

# Default target
all: lua54

# Global clean target
clean: clean-54
	@echo "Cleaned all build artifacts"

# Help target
help:
	@echo "Available targets:"
	@echo "  lua54    - Build Lua 5.4 dynamic library for testing"
	@echo "  clean       - Remove build artifacts and .lua directory"
	@echo "  help        - Show this help message"
