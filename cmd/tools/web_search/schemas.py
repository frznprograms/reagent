from typing import Optional

from pydantic import BaseModel, Field


class SearchInputSchema(BaseModel):
    query: str
    search_depth: str = "advanced"  # for research
    topic: str = "general"
    start_date: Optional[str] = None  # YYYY-MM-DD
    end_date: Optional[str] = None  # YYYY-MM-DD
    max_results: int = 10  # 0–20
    include_images: bool = True
    include_image_descriptions: bool = True  # fixed typo: was "inlcude_"
    include_answer: bool = True
    include_domains: list[str] = Field(default_factory=list)
    exclude_domains: list[str] = Field(default_factory=list)
    country: Optional[str] = None
    timeout: float = 60.0
    include_usage: bool = True
    exact_match: bool = False


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
