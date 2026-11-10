import React from "react";
import { StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { useTheme } from "../theme/ThemeProvider";
import { mediaSource } from "../api/client";
import { Text } from "./Text";

type AvatarProps = {
  name: string;
  photo?: { id: string; url: string } | null;
  size?: number;
  ring?: boolean;
};

export function Avatar({ name, photo, size = 52, ring }: AvatarProps) {
  const theme = useTheme();
  const style = { width: size, height: size, borderRadius: size / 2 };
  return (
    <View
      accessibilityLabel={`Photo de ${name}`}
      style={[style, styles.wrap, { backgroundColor: theme.colors.surfaceAlt }, ring ? { borderWidth: 2, borderColor: theme.colors.primary } : null]}
    >
      {photo ? (
        <Image cachePolicy="memory-disk" contentFit="cover" recyclingKey={photo.id} source={mediaSource(photo)} style={style} transition={150} />
      ) : (
        <Text tone="muted" variant="heading">
          {name.slice(0, 1).toUpperCase()}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { alignItems: "center", justifyContent: "center", overflow: "hidden" }
});
