import sys
import time
from typing import Iterable
from lucas import workflow, function, Workflow, actor
from lucas.serverless_function import Metadata

from actorc.controller.context import (
    ActorContext,
    ActorFunction,
    ActorExecutor,
    ActorRuntime,
    ActorRuntimeClass,
)
import uuid

import json
import pandas as pd
import torch
from datasets import Dataset
from modelscope import snapshot_download, AutoTokenizer
from transformers import AutoModelForCausalLM, TrainingArguments, Trainer, DataCollatorForSeq2Seq, BitsAndBytesConfig # type: ignore
import os
from peft import LoraConfig, get_peft_model, prepare_model_for_kbit_training, PeftModel
import gc

context = ActorContext.createContext("localhost:8082")

PROMPT = "你是一个医学专家，你需要根据用户的问题，给出带有思考的回答。"
MAX_LENGTH = 1000

train_jsonl_new_path = "/home/spark4862/Documents/projects/go/ignis/clients/demo/datasets"
lora_path = "/home/spark4862/Documents/projects/go/ignis/clients/demo/lora_adapter_checkpoint"

tokenizer = AutoTokenizer.from_pretrained("Qwen/Qwen3-0.6B", use_fast=False, trust_remote_code=True)
tokenizer.padding_side = 'left'

@actor(
    wrapper=ActorRuntimeClass, dependency=[], provider='actor', name='read_data',venv='test2'
)
class ReadData:
    global tokenizer
    tk = tokenizer

    def __call__(self, path: str):

        def process_func(example):
            """
            将数据集进行预处理
            """ 
            input_ids, attention_mask, labels = [], [], []
            instruction = self.tk(
                f"<|im_start|>system\n{PROMPT}<|im_end|>\n<|im_start|>user\n{example['input']}<|im_end|>\n<|im_start|>assistant\n",
                add_special_tokens=False,
            )
            response = self.tk(f"{example['output']}", add_special_tokens=False)
            input_ids = instruction["input_ids"] + response["input_ids"] + [self.tk.pad_token_id]
            # 手动添加结束符
            attention_mask = (
                instruction["attention_mask"] + response["attention_mask"] + [1]
            )
            labels = [-100] * len(instruction["input_ids"]) + response["input_ids"] + [self.tk.pad_token_id]
            if len(input_ids) > MAX_LENGTH:  # 做一个截断
                input_ids = input_ids[:MAX_LENGTH]
                attention_mask = attention_mask[:MAX_LENGTH]
                labels = labels[:MAX_LENGTH]
            return {"input_ids": input_ids, "attention_mask": attention_mask, "labels": labels}   

        train_df = pd.read_json(path, lines=True)
        train_ds = Dataset.from_pandas(train_df)
        train_dataset = train_ds.map(process_func, remove_columns=train_ds.column_names)

        print("test read data")

        return train_dataset

@actor(
    wrapper=ActorRuntimeClass, dependency=[], provider='actor', name='train',venv='test2'
)
class Train:
    inited = False

    args = None
    dc = None
    mp = None

    @classmethod
    def initOnce(cls):
        if cls.inited:
            return
        cls.inited = True
        

        cls.args = TrainingArguments(
            # output_dir="/home/spark4862/Documents/projects/go/ignis/clients/demo/output/Qwen3-0.6B",
            per_device_train_batch_size=1,
            gradient_accumulation_steps=4,
            eval_strategy="no",
            logging_steps=1, # logging可以保留，用于观察loss
            save_steps=100000, # 防止Trainer自动保存
            learning_rate=1e-2,
            save_on_each_node=True,
            gradient_checkpointing=True,
            run_name="qwen3-0.6B-manual-loop",
            optim="paged_adamw_8bit",
        )

        bnb_config = BitsAndBytesConfig(
            load_in_4bit=True,  # 启用4-bit量化
            bnb_4bit_quant_type="nf4",  # 设置量化类型，"nf4"是推荐的默认值
            bnb_4bit_compute_dtype=torch.bfloat16,  # 设置计算时的数据类型，以保持精度和速度
            bnb_4bit_use_double_quant=True,  # 启用双重量化，进一步节省显存
        )

        lora_config = LoraConfig(
            r=16,
            lora_alpha=32,
            target_modules="all-linear",
            lora_dropout=0.05,
            bias="none",
            task_type="CAUSAL_LM",
        )

        cls.dc = DataCollatorForSeq2Seq(tokenizer=tokenizer, padding=True, pad_to_multiple_of=8)

        model = AutoModelForCausalLM.from_pretrained(
            "Qwen/Qwen3-0.6B", 
            device_map="auto", 
            torch_dtype=torch.float16, 
            quantization_config=bnb_config,
            attn_implementation="flash_attention_2"
        )
        if not os.path.exists(lora_path):
            modelP = get_peft_model(model, lora_config)
        else:
            print("from pretrained", file=sys.stderr)
            modelP = PeftModel.from_pretrained(model, lora_path)
        modelP.gradient_checkpointing_enable() # type: ignore
        modelP.enable_input_require_grads() # type: ignore

        cls.mp = modelP


    def __call__(self, ds: Dataset):
        print("test train")
        self.initOnce()

        trainer = Trainer(
            model=self.mp,
            args=self.args, #type: ignore
            train_dataset=ds,
            data_collator=self.dc,
        )
        trainer.train()
        self.mp.save_pretrained(lora_path)

        return "test"

# def predict(path: str):
#     # tokenizer = AutoTokenizer.from_pretrained("Qwen/Qwen3-0.6B", use_fast=False, trust_remote_code=True)
#     # tokenizer.padding_side = 'left'
#     # global tokenizer
#     bnb_config = BitsAndBytesConfig(
#         load_in_4bit=True,  # 启用4-bit量化
#         bnb_4bit_quant_type="nf4",  # 设置量化类型，"nf4"是推荐的默认值
#         bnb_4bit_compute_dtype=torch.bfloat16,  # 设置计算时的数据类型，以保持精度和速度
#         bnb_4bit_use_double_quant=True,  # 启用双重量化，进一步节省显存
#     )
#     model1 = AutoModelForCausalLM.from_pretrained(
#         "Qwen/Qwen3-0.6B", 
#         device_map="auto", 
#         torch_dtype=torch.float16, 
#         quantization_config=bnb_config,
#         attn_implementation="flash_attention_2"
#     )
#     # global model
#     model1 = model1
#     if os.path.exists(lora_path):
#         print("test1", file=sys.stderr)
#         model1 = PeftModel.from_pretrained(model1, lora_path)
    
#     test_df = pd.read_json(path, lines=True)
#     for index, row in test_df.iterrows():
#         instruction = row['instruction']
#         input_value = row['input']

#         messages = [
#             {"role": "system", "content": f"{instruction}，请用简洁明了的语言直接回答问题，避免重复、解释性冗余和结构化表达"},
#             # , 简要总结
#             {"role": "user", "content": f"{input_value}"}
#         ]

#         device = "cuda"
#         text = tokenizer.apply_chat_template(
#             messages,
#             tokenize=False,
#             add_generation_prompt=True,
#             # enable_thinking=False
#         )
#         text = text + "<think>\n\n</think>\n\n"
#         model_inputs = tokenizer([text], return_tensors="pt").to(device)
#         generated_ids = model1.generate(
#             model_inputs.input_ids,
#             attention_mask=model_inputs.attention_mask,
#             max_new_tokens=MAX_LENGTH,
#             temperature=0.3,
#             top_p=0.8,
#             top_k=20,
#             min_p=0,
#             repetition_penalty=1.2,
#             penalty_alpha=0.6,
#             do_sample=True,
#             exponential_decay_length_penalty=(80, 1.03),
#         )
#         generated_ids = [
#             output_ids[len(input_ids):] for input_ids, output_ids in zip(model_inputs.input_ids, generated_ids)
#         ]
#         response = tokenizer.batch_decode(generated_ids, skip_special_tokens=True)[0]
#         print(f"""
#         # Question: {input_value}

#         # LLM:{response}
#         # """, file=sys.stderr)
#         # responses.append()
#     model1.cpu()
#     del model1
#     gc.collect()
#     torch.cuda.synchronize()
#     torch.cuda.empty_cache()
#     # return responses


# @workflow(executor=ActorExecutor) # type: ignore
# def workflowfunc(wf: Workflow):
#     _in = wf.input()
#     ReadData.export()
#     Train.export()
    
#     ds = wf.call("read_data", {"path": _in["path"]})
#     wf.call("train", {"ds": ds})
#     return "test"

@workflow(executor=ActorExecutor) # type: ignore
def workflowfunc(wf: Workflow):
    _in = wf.input()
    
    ReadData.export()
    Train.export()

    paths = wf.catch(_in["paths"])
    ds_list = paths.map(lambda p: wf.call("read_data", {"path": p}))
    _ = ds_list.map(lambda ds: wf.call("train", {"ds": ds}))
    return "test"

def actorWorkflowExportFunc(dict: dict):

    # just use for local invoke
    from lucas import routeBuilder
    route = routeBuilder.build()
    route_dict = {}
    for function in route.functions:
        route_dict[function.name] = function.handler
    for workflow in route.workflows:
        route_dict[workflow.name] = workflow._generate_workflow
    for actor in route.actors:
        route_dict[actor.name] = actor._cls
    metadata = Metadata(
        id=str(uuid.uuid4()),
        params=dict,
        namespace=None,
        router=route_dict,
        request_type="invoke",
        redis_db=None,
        producer=None
    )
    rt = ActorRuntime(metadata)
    workflowfunc.set_runtime(rt)
    workflow = workflowfunc.generate()
    return workflow.execute()


workflow_func = workflowfunc.export(actorWorkflowExportFunc)
print("----first execute----")
data_dir = "/home/spark4862/Documents/projects/go/ignis/clients/demo/.bak/"

# predict("/home/spark4862/Documents/projects/go/ignis/clients/demo/datasets/medical41.jsonl")

# for file in os.listdir(data_dir):
#     workflow_func({"path": os.path.join(data_dir, file)})

workflow_func({"paths": list(map(lambda f: os.path.join(data_dir, f), os.listdir(data_dir)))})

# predict("/home/spark4862/Documents/projects/go/ignis/clients/demo/datasets/medical41.jsonl")
