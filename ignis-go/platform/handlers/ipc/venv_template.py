from argparse import ArgumentParser

from actorc.executor import Executor


def parse_args():
    parser = ArgumentParser()
    parser.add_argument("--ipc", type=str, required=True)
    parser.add_argument("--venv", type=str, required=True)
    return parser.parse_args()


def main():
    args = parse_args()
    executor = Executor(args.venv)
    executor.serve(args.ipc)


if __name__ == "__main__":
    main()
