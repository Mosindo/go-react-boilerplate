import React from "react";
import { StyleSheet } from "react-native";
import { PhotoImage } from "../media/PhotoImage";
import { controls } from "./tokens";

type AvatarProps = {
  path?: string | null;
  name?: string;
  size?: keyof typeof controls.avatar;
};

export function Avatar({ path, name, size = "md" }: AvatarProps) {
  const dimension = controls.avatar[size];
  return (
    <PhotoImage
      accessibilityLabel={name ? `Photo de ${name}` : undefined}
      fallbackLabel={name}
      path={path}
      style={[styles.round, { width: dimension, height: dimension, borderRadius: dimension / 2 }]}
    />
  );
}

const styles = StyleSheet.create({ round: { overflow: "hidden" } });
