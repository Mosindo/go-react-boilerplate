import React from "react";
import { Image, StyleSheet, View, type StyleProp, type ViewStyle } from "react-native";
import { useAuthedImage } from "../useAuthedImage";
import { Text } from "./Text";
import { useTheme } from "./theme";
import { controls } from "./tokens";

export type AvatarProps = {
  name?: string;
  size?: number;
  /** Server-relative photo path such as /photos/<id>/content. */
  photoPath?: string | null;
  style?: StyleProp<ViewStyle>;
};

export function Avatar({ name, size = controls.avatar.md, photoPath, style }: AvatarProps) {
  const { colors } = useTheme();
  const source = useAuthedImage(photoPath);
  return (
    <View
      accessibilityLabel={name}
      style={[
        styles.root,
        { width: size, height: size, borderRadius: size / 2, backgroundColor: colors.primarySoft, borderColor: colors.primaryBorder },
        style
      ]}
    >
      {source ? (
        <Image source={source} style={styles.image} />
      ) : (
        <Text style={{ fontSize: size * 0.38 }} tone="primary" weight="bold">
          {(name?.trim()[0] ?? "?").toUpperCase()}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  root: { alignItems: "center", justifyContent: "center", overflow: "hidden", borderWidth: 1 },
  image: { width: "100%", height: "100%" }
});
