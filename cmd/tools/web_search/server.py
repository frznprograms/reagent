import json
import os
from datetime import datetime, timezone

from dotenv import load_dotenv
from fastmcp import FastMCP
from fastmcp.exceptions import ToolError
from tavily import TavilyClient  # for async, use AsyncTavilyClient

from .schemas import SearchInputSchema, SearchResponseSchema

load_dotenv()


mcp = FastMCP(name="Web Search", strict_input_validation=True)


def init_search_client():
    tavily_api_key = os.getenv("TAVILY_API_KEY")
    if not tavily_api_key:
        raise ValueError("Please set a Tavily API Key in the environment.")
    client = TavilyClient(tavily_api_key)
    return client


@mcp.tool(
    name="WebSearch",
    description=(
        "Search the web; obtain relevant results for a given query.",
        "If a destination ('dest') key is set, then the results should be saved to it.",
        "If not, the default behaviour is to return the search results to the LLM.",
    ),
)
def web_search(input: SearchInputSchema) -> str:
    client = init_search_client()
    kwargs = input.model_dump(exclude_none=True)
    query = kwargs.pop("query")
    if not query:
        raise ToolError("A valid query must be provided.")

    raw_response = client.search(query=query, **kwargs)
    response = SearchResponseSchema(**raw_response)

    dest = kwargs.pop("dest")
    if dest:
        os.makedirs(os.path.dirname(os.path.abspath(dest)), exist_ok=True)
        payload = {
            "query": response.query,
            "answer": response.answer,
            "results": [r.model_dump() for r in response.results],
            "images": [
                i.model_dump()  # type: ignore
                if hasattr(i, "model_dump")
                else i
                for i in response.images
            ],
            "saved_at": datetime.now(timezone.utc).isoformat(),
        }
        with open(dest, "w") as f:
            json.dump(payload, f, indent=4)
        return dest

    return json.dumps(
        {
            "query": response.query,
            "answer": response.answer,
            "results": [r.model_dump() for r in response.results],
        }
    )
