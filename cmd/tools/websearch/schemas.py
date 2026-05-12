from typing import Literal, Optional

from pydantic import BaseModel, Field


class SearchInputSchema(BaseModel):
    query: str = Field(description="The search query to run.")
    dest: str | None = Field(
        default=None,
        description="Optional file path to save results as JSON. If omitted, results are returned directly.",
    )
    search_depth: Literal["basic", "advanced"] | None = Field(
        default=None,
        description="'basic' is faster; 'advanced' is slower but more thorough.",
    )
    topic: Literal["general", "news", "finance"] | None = Field(
        default=None,
        description="'general' for broad searches, 'news' for recent events, 'finance' for market data.",
    )
    time_range: Literal["day", "week", "month", "year"] | None = Field(
        default=None, description="Filter results by recency."
    )
    max_results: int | None = Field(
        default=None, description="Number of results to return, typically 1-10."
    )
    include_images: bool = Field(
        default=True,
        description="If true, image URLs relevant to the query are included.",
    )
    include_answer: bool = Field(
        default=True,
        description="If true, Tavily returns a synthesised answer string above the results.",
    )
    include_domains: list[str] = Field(
        default_factory=list,
        description="Restrict results to these domains e.g. ['github.com', 'docs.python.org'].",
    )
    exclude_domains: list[str] = Field(
        default_factory=list, description="Block results from these domains."
    )
    country: str | None = Field(
        default=None,
        description="Bias results toward a specific country code e.g. 'us', 'gb'.",
    )
    timeout: float = Field(default=60.0, description="Request timeout in seconds.")


class ImageResult(BaseModel):
    url: str
    description: str


class SearchResult(BaseModel):
    title: str
    url: str
    content: str
    score: float
    raw_content: Optional[str] = None
    published_date: Optional[str] = None
    favicon: Optional[str] = None
    images: list[str] | list[ImageResult] = []


class SearchResponseSchema(BaseModel):
    results: list[SearchResult]
    query: str
    response_time: float
    answer: str = ""
    images: list[str] | list[ImageResult] = []
    request_id: str = ""
