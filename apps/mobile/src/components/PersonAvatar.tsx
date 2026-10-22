import React from "react";
import type { PhotoRef } from "../api/types";
import { AuthImage } from "./AuthImage";

type PersonAvatarProps = { name: string; photo?: PhotoRef | null; size?: number };

/** Round avatar from a profile photo, falling back to the first letter of the name. */
export function PersonAvatar({ name, photo, size = 48 }: PersonAvatarProps) {
  return (
    <AuthImage
      label={`Photo de ${name}`}
      photo={photo}
      placeholder={name.slice(0, 1).toUpperCase()}
      size="thumb"
      style={{ width: size, height: size, borderRadius: size / 2 }}
    />
  );
}
