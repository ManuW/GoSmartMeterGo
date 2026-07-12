# Makefile für GoSmartMeterGo
# Unterstützt das Kompilieren für macOS (lokal) und Raspberry Pi (Linux 32-Bit ARMv7 und 64-Bit ARM64)

# Variablen
BINARY_NAME=gosmartmeter
SRC=main.go

.PHONY: all build build-local build-rpi-32 build-rpi-64 clean

# Standard-Target: Kompiliert für alle Plattformen
all: build-local build-rpi-32 build-rpi-64

# Kompiliert für das aktuelle Betriebssystem (z.B. macOS)
build: build-local

build-local:
	@echo "Kompiliere für lokales Betriebssystem..."
	go build -o $(BINARY_NAME) $(SRC)

# Kompiliere für Raspberry Pi 2 / 3 (Linux 32-Bit ARMv7)
# GOARM=7 optimiert für den Cortex-A7 Prozessor des Raspi 2
build-rpi-32:
	@echo "Kompiliere für Raspberry Pi 2/3 (Linux ARMv7 32-Bit)..."
	GOOS=linux GOARCH=arm GOARM=7 go build -o $(BINARY_NAME)-rpi-32 $(SRC)

# Kompiliere für neuere Raspberries oder 64-Bit OS (Linux ARM64)
build-rpi-64:
	@echo "Kompiliere für Raspberry Pi 3/4/5 (Linux ARM64 64-Bit)..."
	GOOS=linux GOARCH=arm64 go build -o $(BINARY_NAME)-rpi-64 $(SRC)

# Bereinigen der Kompilate und Pakete
clean:
	@echo "Bereinige Binärdateien und Pakete..."
	rm -f $(BINARY_NAME) $(BINARY_NAME)-rpi-32 $(BINARY_NAME)-rpi-64
	rm -f $(BINARY_NAME)-rpi-32.deb $(BINARY_NAME)-rpi-64.deb

.PHONY: pack-rpi-32 pack-rpi-64 pack-all

# Erstelle .deb Paket für Raspberry Pi 32-Bit (armhf)
pack-rpi-32: build-rpi-32
	@echo "Erstelle .deb Paket für Raspberry Pi 32-Bit (armhf)..."
	cd packaging && ARCH=armhf BINARY_SOURCE=$(BINARY_NAME)-rpi-32 nfpm pkg --target ../$(BINARY_NAME)-rpi-32.deb

# Erstelle .deb Paket für Raspberry Pi 64-Bit (arm64)
pack-rpi-64: build-rpi-64
	@echo "Erstelle .deb Paket für Raspberry Pi 64-Bit (arm64)..."
	cd packaging && ARCH=arm64 BINARY_SOURCE=$(BINARY_NAME)-rpi-64 nfpm pkg --target ../$(BINARY_NAME)-rpi-64.deb

# Erstelle alle Pakete
pack-all: pack-rpi-32 pack-rpi-64
