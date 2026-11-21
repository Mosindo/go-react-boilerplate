import React from "react";
import { Image, type ImageProps } from "expo-image";
import { absoluteUrl } from "../api/config";

type RemoteImageProps = Omit<ImageProps, "source"> & { uri: string | null | undefined };

/** Displays an API-hosted image (signed photo URLs are relative to the API). */
export function RemoteImage({ uri, ...props }: RemoteImageProps) {
  return (
    <Image source={uri ? { uri: absoluteUrl(uri) } : undefined} cachePolicy="memory-disk" transition={180} contentFit="cover" {...props} />
  );
}

export function prefetchImages(uris: string[]): void {
  if (uris.length > 0) {
    void Image.prefetch(uris.map(absoluteUrl));
  }
}
