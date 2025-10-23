"""Configuration management for Critica."""

import json
import os
from dataclasses import dataclass, field
from pathlib import Path
from typing import Optional


@dataclass
class Config:
    """Configuration for Critica AI features."""

    # AI Configuration
    ai_enabled: bool = True
    openai_api_key: Optional[str] = None
    openai_model: str = "gpt-5-nano"
    openai_base_url: str = "https://api.openai.com/v1"


def get_default_config_path() -> Path:
    """Get the default configuration file path."""
    if os.name == "nt":  # Windows
        config_dir = Path(os.environ.get("APPDATA", Path.home() / "AppData" / "Roaming"))
    else:  # Unix-like
        config_dir = Path(os.environ.get("XDG_CONFIG_HOME", Path.home() / ".config"))

    return config_dir / "critica" / "config.json"


def load_config() -> Config:
    """Load configuration from file and environment variables.

    Priority (highest to lowest):
    1. Environment variables
    2. Config file
    3. Default values
    """
    config = Config()

    # Try to load from config file
    config_path = get_default_config_path()
    if config_path.exists():
        try:
            with open(config_path, "r") as f:
                data = json.load(f)

            # Update config with file values
            if "ai_enabled" in data:
                config.ai_enabled = bool(data["ai_enabled"])
            if "openai_api_key" in data:
                config.openai_api_key = data["openai_api_key"]
            if "openai_model" in data:
                config.openai_model = data["openai_model"]
            if "openai_base_url" in data:
                config.openai_base_url = data["openai_base_url"]
        except (json.JSONDecodeError, IOError) as e:
            # Silently ignore config file errors and use defaults
            pass

    # Override with environment variables (highest priority)
    if os.environ.get("OPENAI_API_KEY"):
        config.openai_api_key = os.environ["OPENAI_API_KEY"]
    if os.environ.get("OPENAI_MODEL"):
        config.openai_model = os.environ["OPENAI_MODEL"]
    if os.environ.get("OPENAI_BASE_URL"):
        config.openai_base_url = os.environ["OPENAI_BASE_URL"]

    return config


def save_config(config: Config) -> None:
    """Save configuration to file."""
    config_path = get_default_config_path()
    config_path.parent.mkdir(parents=True, exist_ok=True)

    data = {
        "ai_enabled": config.ai_enabled,
        "openai_model": config.openai_model,
        "openai_base_url": config.openai_base_url,
    }

    # Only save API key if it was set in the config (not from env)
    if config.openai_api_key and not os.environ.get("OPENAI_API_KEY"):
        data["openai_api_key"] = config.openai_api_key

    with open(config_path, "w") as f:
        json.dump(data, f, indent=2)
