BINARY_NAME = minh-kit
INSTALL_PATH = $(HOME)/local/bin

install:
	@echo "🚧 Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) minh-kit.go

	@echo "📁 Creating install directory if not exists..."
	mkdir -p $(INSTALL_PATH)

	@echo "📦 Installing to $(INSTALL_PATH)/$(BINARY_NAME)..."
	mv $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	chmod +x $(INSTALL_PATH)/$(BINARY_NAME)

	@echo "🔁 Adding $(INSTALL_PATH) to PATH if missing..."
	grep -qxF 'export PATH="$(INSTALL_PATH):$$PATH"' ~/.zshrc || echo 'export PATH="$(INSTALL_PATH):$$PATH"' >> ~/.zshrc

	@echo "✅ Installed as $(BINARY_NAME). Restart terminal or run: source ~/.zshrc"

clean:
	@echo "🧹 Cleaning up build files..."
	rm -f $(BINARY_NAME)