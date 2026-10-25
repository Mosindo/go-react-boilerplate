import type { UploadFile } from "../../api";

type PickedAsset = { uri: string; fileName?: string | null; mimeType?: string | null };

/** Builds the multipart descriptor; the API only accepts JPEG or PNG. */
export function toUploadFile(asset: PickedAsset): UploadFile {
  const isPng = asset.mimeType === "image/png" || asset.uri.toLowerCase().endsWith(".png");
  const extension = isPng ? "png" : "jpg";
  const baseName = (asset.fileName ?? "photo").replace(/\.[^.]+$/, "") || "photo";
  return { uri: asset.uri, name: `${baseName}.${extension}`, type: isPng ? "image/png" : "image/jpeg" };
}
