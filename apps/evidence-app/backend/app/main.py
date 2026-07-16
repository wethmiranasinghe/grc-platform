import io
import mimetypes

from azure.storage.blob import BlobServiceClient
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import StreamingResponse

from app.api.routes import products, frameworks, controls, evidence, submissions, agent, usage, me
from app.config import settings

app = FastAPI(title="Compliance Evidence Portal", version="1.0.0", redirect_slashes=False)

cors_origins = [o.strip() for o in settings.CORS_ORIGINS.split(",") if o.strip()]

app.add_middleware(
    CORSMiddleware,
    allow_origins=cors_origins,
    allow_methods=["*"],
    allow_headers=["*"],
)

_storage_client = BlobServiceClient.from_connection_string(
    settings.AZURE_STORAGE_CONNECTION_STRING
)


@app.get("/uploads/{filename}")
def serve_blob_file(filename: str):
    blob = _storage_client.get_blob_client(
        container=settings.AZURE_STORAGE_CONTAINER, blob=filename
    )
    data = blob.download_blob().readall()
    content_type = mimetypes.guess_type(filename)[0] or "application/octet-stream"
    return StreamingResponse(io.BytesIO(data), media_type=content_type)


app.include_router(products.router, prefix="/api")
app.include_router(frameworks.router, prefix="/api")
app.include_router(controls.router, prefix="/api")
app.include_router(evidence.router, prefix="/api")
app.include_router(submissions.router, prefix="/api")
app.include_router(agent.router, prefix="/api")
app.include_router(usage.router, prefix="/api")
app.include_router(me.router, prefix="/api")


@app.get("/health")
def health_check():
    return {"status": "ok"}
