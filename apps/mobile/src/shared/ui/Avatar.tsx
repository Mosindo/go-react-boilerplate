import React from "react";
import { StyleSheet, View } from "react-native";
import { useTheme } from "../../theme";
import { PhotoImage } from "./Photo";
import { Text } from "./Text";
import { controls } from "./tokens";

type AvatarProps = { name: string; path?: string; size?: keyof typeof controls.avatar };

export function Avatar({ name, path, size = "md" }: AvatarProps) {
  const { colors } = useTheme();
  const dimension = controls.avatar[size];
  const radius = dimension / 2;
  if (path) return <PhotoImage label={name} path={path} style={{ width: dimension, height: dimension, borderRadius: radius }} />;
  return (
    <View
      accessibilityLabel={name}
      style={[styles.fallback, { width: dimension, height: dimension, borderRadius: radius, backgroundColor: colors.primarySoft }]}
    >
      <Text tone="primary" variant="label">
        {name.trim().charAt(0).toUpperCase()}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  fallback: { alignItems: "center", justifyContent: "center" }
});
