from datetime import datetime
from pydantic import BaseModel


class EvidenceCreate(BaseModel):
    title: str
    description: str | None = None
    file_name: str
    file_url: str
    control_id: int


class EvidenceFileOut(BaseModel):
    id: int
    file_name: str
    file_url: str
    subtask: str | None

    model_config = {"from_attributes": True}


class EvidenceResponse(BaseModel):
    id: int
    title: str
    description: str | None
    file_name: str
    file_url: str
    control_id: int | None
    created_by: str
    created_at: datetime
    updated_at: datetime
    files: list[EvidenceFileOut] = []

    model_config = {"from_attributes": True}
