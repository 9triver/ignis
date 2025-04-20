from setuptools import find_packages, setup

setup(
    name="actorc",
    version="1.0",
    packages=find_packages(),
    install_requires=["httpx", "pyzmq", "protobuf"],
    author="Tianqi Ren",
    description="Description of your package",
    license="Apache 2.0",
)
