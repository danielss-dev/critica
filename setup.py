"""Setup script for Critica."""

from setuptools import setup, find_packages

setup(
    name="critica",
    version="2.0.0",
    packages=find_packages(),
    install_requires=[
        "openai>=1.54.0",
        "click>=8.1.0",
        "rich>=13.0.0",
    ],
    entry_points={
        "console_scripts": [
            "critica=critica_py.cli:main",
        ],
    },
    python_requires=">=3.9",
)
