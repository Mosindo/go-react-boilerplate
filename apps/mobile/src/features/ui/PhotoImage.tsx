import React from "react";
import { Image, type ImageStyle, type StyleProp } from "react-native";
import { API_BASE_URL } from "../../api/client";
import { resolveMediaUrl } from "../media";
import type { Photo } from "../types";

export function photoUri(photo: Photo | null | undefined): string | null {
  return photo ? resolveMediaUrl(API_BASE_URL, photo.url) : null;
}

type Props = {
  photo: Photo | null | undefined;
  style: StyleProp<ImageStyle>;
  accessibilityLabel?: string;
};

export function PhotoImage({ photo, style, accessibilityLabel }: Props) {
  const uri = photoUri(photo);
  if (!uri) {
    return null;
  }
  return (
    <Image
      accessibilityLabel={accessibilityLabel}
      accessible={accessibilityLabel !== undefined}
      resizeMode="cover"
      source={{ uri }}
      style={style}
    />
  );
}
