from actorc.utils import serde


class Add:
    def call(self, a, b):
        return a + b


with open("func.pkl", "wb") as f:
    serde.dump(Add(), f)
