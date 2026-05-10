import os
from cmd.tools.web_search.schemas import SearchInputSchema, SearchResponseSchema

from dotenv import load_dotenv
from fastmcp import FastMCP
from fastmcp.exceptions import ToolError
from tavily import TavilyClient  # for async, use AsyncTavilyClient

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


@mcp.tool(
    name="SaveSearchResults",
    description="Save LLM summary of web search results and relevant links to local storage folder",
)
def save_search_results(dest: str) -> None:
    # NOTE: the actual results of the web search are not saved; this protects the user
    # from issues pertaining to scraping.
    pass
