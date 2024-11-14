import asyncio

from actorc.dag import IR, function, workflow, instance
from actorc.utils import PlatformEndPoint, EncDec
from deep_river.regression import RollingRegressor
from river import compose, preprocessing
from torch import nn

ep = PlatformEndPoint("localhost", 2313)


class RegressionNet(nn.Module):
    def __init__(self, n_features, hidden_size):
        super().__init__()
        self.n_features = n_features
        self.embedding = nn.Linear(n_features, hidden_size)
        self.transformer = nn.Transformer(
            d_model=hidden_size,
            nhead=8,
            num_encoder_layers=16,
            batch_first=True,
        )
        self.fc = nn.Linear(in_features=hidden_size, out_features=1)

    def forward(self, x):
        x = self.embedding(x)
        x = self.transformer(x, x)
        x = self.fc(x[-1])
        return x


def get_model():
    regressor = RollingRegressor(
        module=RegressionNet,
        loss_fn="mse",
        optimizer_fn="sgd",
        window_size=64,
        lr=1e-3,
        hidden_size=128,  # parameters of MyModule can be overwritten
        append_predict=True,
        device="cuda:0",
    )

    model = (
        compose.Select(
            "date", "HUFL", "HULL", "MUFL", "MULL", "LUFL", "LULL"  # type:ignore
        )
        | preprocessing.StandardScaler()
        | regressor
    )

    return model, regressor.module


@function()
def collect(train):
    x = {
        "date": 1467345600.0,
        "HUFL": 5.827000141143799,
        "HULL": 2.009000062942505,
        "MUFL": 1.5989999771118164,
        "MULL": 0.4620000123977661,
        "LUFL": 4.203000068664552,
        "LULL": 1.3400000333786009,
    }
    y = 30.5310001373291

    if train:
        return {**x, "OT": y}
    return x


@instance()
class Model:
    def __init__(self) -> None:
        self.model, self.torch_module = get_model()

    def update(self, info):
        print(info)
        value = info.get("OT", None)
        self.model.learn_one(info, value)
        return "ok"

    def predict(self, info):
        print(info)
        result = self.model.predict_one(info)
        return result


model = Model()


@workflow([("model", model)])
def learn(ir: IR):
    info = ir.func(collect, {"train": True})
    res = ir.func(model.update, {"info": info})
    return res


@workflow([("model", model)])
def inference(ir: IR):
    info = ir.func(collect, {"train": False})
    res = ir.func(model.predict, {"info": info})
    return res


def parse_result(result):
    if result["status"] != 200:
        return
    resp = result["result"]
    for node, obj in resp["Results"].items():
        ret = EncDec.decode_dict(obj)
        print(f"Node '{node}' returns '{ret}'")


async def main():
    train = learn()
    await train.deploy("train-dag", ep)

    predict = inference()
    await predict.deploy("predict-dag", ep)

    result = await predict.execute_remote("predict-dag", ep)
    parse_result(result)

    for _ in range(100):
        result = await train.execute_remote("train-dag", ep)

    result = await predict.execute_remote("predict-dag", ep)
    parse_result(result)


if __name__ == "__main__":
    asyncio.run(main())
