import React from "react";
import { Image, type ImageStyle, type StyleProp } from "react-native";
import { absoluteUrl, mediaHeaders } from "../api/client";
import { usePalette } from "../theme";

type Props = { path?: string; style?: StyleProp<ImageStyle>; label?: string };

/** Photos are private: they are fetched with the session's bearer token. */
export function AuthImage({ path, style, label }: Props) {
  const p = usePalette();
  if (!path) {
    return <Image accessibilityLabel={label} style={[{ backgroundColor: p.surfaceAlt }, style]} />;
  }
  return (
    <Image
      accessibilityLabel={label}
      source={{ uri: absoluteUrl(path), headers: mediaHeaders() }}
      style={[{ backgroundColor: p.surfaceAlt }, style]}
      resizeMode="cover"
    />
  );
}

export function Avatar({
  photoId,
  size = 48,
  label
}: {
  photoId?: string;
  size?: number;
  label?: string;
}) {
  return (
    <AuthImage
      path={photoId ? `/photos/${photoId}` : undefined}
      label={label}
      style={{ width: size, height: size, borderRadius: size / 2 }}
    />
  );
}
