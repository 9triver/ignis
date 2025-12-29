import abc
from typing import Any, TypedDict


class RetrievalResult(TypedDict):
    backend_name: str
    documents: list[dict[str, Any]]
    retrieval_time: float


class LLMResponse(TypedDict):
    answer: str
    used_documents: list[dict[str, Any]]
    generation_time: float


class VectorDBBackend(abc.ABC):
    @abc.abstractmethod
    def get_name(self) -> str:
        pass

    @abc.abstractmethod
    def retrieve(self, query: list[int], top_k: int = 5) -> RetrievalResult:
        pass


class LLMClient(abc.ABC):
    @abc.abstractmethod
    def generate(
        self,
        query: str,
        retrieval_results: list[RetrievalResult],
    ) -> LLMResponse:
        pass


class EmbeddingClient:
    @abc.abstractmethod
    def encode(self, query: str) -> list[int]:
        pass


class MockVectorDB(VectorDBBackend):
    def __init__(self, name: str) -> None:
        self.name = name

    def get_name(self) -> str:
        return self.name

    def retrieve(self, query: list[int], top_k: int = 5) -> RetrievalResult:
        import time

        start = time.time()
        mock_docs = [
            {
                "id": f"{self.name}_{i}",
                "content": f"Mock document from ChromaDB for query",
                "similarity": round(0.9 - i * 0.05, 2),
            }
            for i in range(top_k)
        ]
        time.sleep(3.0)
        retrieval_time = round(time.time() - start)

        return RetrievalResult(
            backend_name=self.get_name(),
            documents=mock_docs,
            retrieval_time=retrieval_time,
        )


class MockLLMClient(LLMClient):
    def generate(
        self,
        query: str,
        retrieval_results: list[RetrievalResult],
    ) -> LLMResponse:
        import time

        start = time.time()

        all_docs = []
        for result in retrieval_results:
            all_docs.extend(result["documents"])

        answer = f"Based on your query: {query}, here's the answer using {len(all_docs)} retrieved documents from {[r['backend_name'] for r in retrieval_results]}."

        generation_time = round(time.time() - start, 4)
        return LLMResponse(
            answer=answer,
            used_documents=all_docs,
            generation_time=generation_time,
        )


class MockEmbeddingClient(EmbeddingClient):
    def encode(self, query: str) -> list[int]:
        return [i + ord(ch) for i, ch in enumerate(query)]


class RAGApplication:
    def __init__(
        self,
        vector_db_backends: list[VectorDBBackend],
        embedding_client: EmbeddingClient,
        llm_client: LLMClient,
    ):
        self.vector_db_backends = vector_db_backends
        self.embedding_client = embedding_client
        self.llm_client = llm_client

    def process_input(self, raw_input: str) -> str:
        processed_text = raw_input.strip().lower()
        return processed_text

    def retrieve_documents(self, query: str) -> list[RetrievalResult]:
        retrieval_results = []
        q_embed = self.embedding_client.encode(query)
        for backend in self.vector_db_backends:
            print(f"retrieving documents from {backend.get_name()}")
            result = backend.retrieve(q_embed)
            retrieval_results.append(result)

        return retrieval_results

    def generate_answer(
        self,
        query: str,
        retrieval_results: list[RetrievalResult],
    ) -> LLMResponse:
        return self.llm_client.generate(query, retrieval_results)

    def run(self, raw_input: str) -> LLMResponse:
        query = self.process_input(raw_input)
        retrieval_results = self.retrieve_documents(query)
        llm_response = self.generate_answer(query, retrieval_results)
        return llm_response


if __name__ == "__main__":
    vector_db_backends: list[VectorDBBackend] = [
        MockVectorDB("db-1"),
        MockVectorDB("db-2"),
        MockVectorDB("db-3"),
        MockVectorDB("db-4"),
    ]
    llm_client = MockLLMClient()
    embedding_client = MockEmbeddingClient()
    rag_app = RAGApplication(vector_db_backends, embedding_client, llm_client)

    test_query = "Test Query"
    response = rag_app.run(test_query)

    print(f"Final Answer: {response['answer']}")
