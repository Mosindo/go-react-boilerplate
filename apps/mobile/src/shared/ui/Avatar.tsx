import React from "react";
import { StyleSheet } from "react-native";
import { AuthImage } from "./AuthImage";

type Props = {
  thumbUrl: string | null | undefined;
  size?: number;
  name?: string;
};

export const Avatar = React.memo(function Avatar({ thumbUrl, size = 48, name }: Props) {
  return (
    <AuthImage
      accessibilityLabel={name ? `Photo de ${name}` : undefined}
      path={thumbUrl}
      style={[styles.image, { width: size, height: size, borderRadius: size / 2 }]}
    />
  );
});

const styles = StyleSheet.create({ image: { overflow: "hidden" } });
