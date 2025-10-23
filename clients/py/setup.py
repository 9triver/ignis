from setuptools import find_packages, setup

setup(
    name="actorc",
    version="1.0",
    packages=find_packages(),
    install_requires=[
        "httpx", "pyzmq", 
        "grpcio==1.71.0", "protobuf>=3.12.0", 
        "cloudpickle"],
    author="Tianqi Ren",
    description="Description of your package",
    license="Apache 2.0",
)