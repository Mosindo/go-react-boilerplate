import React from "react";
import { Image, type ImageProps } from "expo-image";
import { absoluteUrl } from "../api/config";

/** Stable cache key: the storage key inside a signed URL, which survives URL re-signing. */
export function mediaCacheKey(url: string): string {
  const match = /\/media\/([a-f0-9]+\.jpg)/.exec(url);
  return match ? match[1] : url;
}

type RemoteImageProps = Omit<ImageProps, "source"> & { url: string };

export function RemoteImage({ url, ...props }: RemoteImageProps) {
  return (
    <Image
      cachePolicy="memory-disk"
      contentFit="cover"
      recyclingKey={mediaCacheKey(url)}
      source={{ uri: absoluteUrl(url), cacheKey: mediaCacheKey(url) }}
      transition={180}
      {...props}
    />
  );
}
