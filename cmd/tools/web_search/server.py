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
    description="Search the web; obtain relevant results for a given query",
)
def web_search(input: SearchInputSchema) -> SearchResponseSchema:
    client = init_search_client()
    kwargs = input.model_dump(exclude_none=True)
    query = kwargs.pop("query")
    if not query:
        raise ToolError("A valid query must be provided.")

    raw_response = client.search(query=query, **kwargs)
    return SearchResponseSchema(**raw_response)


# TODO: check what LLM is actually returning; may be useful to concatenate output
# from Tavily instead of using tokens to generate summary; can use the analysis tool
# separately on the output from Tavily AI search.
@mcp.tool(
    name="SaveSearchResults",
    description="Save your summary of web search results and their source URLs to a JSON file at the given path",
)
def save_search_results(
    dest: str, query: str, summary: str, sources: list[str]
) -> None:
    os.makedirs(os.path.dirname(os.path.abspath(dest)), exist_ok=True)
    payload = {
        "query": query,
        "summary": summary,
        "sources": sources,
        "saved_at": datetime.now(timezone.utc).isoformat(),
    }
    with open(dest, "w") as f:
        json.dump(payload, f, indent=4)
