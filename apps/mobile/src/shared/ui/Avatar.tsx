import { Image } from "expo-image";
import React from "react";
import { StyleSheet, View } from "react-native";
import { absoluteUrl } from "../../api/client";
import { useTheme } from "../../theme";
import { Text } from "./Text";

type Props = { name: string; uri?: string | null; size?: number };

export function Avatar({ name, uri, size = 48 }: Props) {
  const theme = useTheme();
  const box = { width: size, height: size, borderRadius: size / 2, backgroundColor: theme.primarySoft };
  if (uri) {
    return (
      <Image
        accessibilityLabel={`Photo de ${name}`}
        source={{ uri: absoluteUrl(uri) }}
        style={box}
        contentFit="cover"
        transition={150}
        cachePolicy="memory-disk"
      />
    );
  }
  return (
    <View style={[box, styles.center]} accessibilityLabel={`Avatar de ${name}`}>
      <Text variant="label" tone="primary" style={{ fontSize: size / 2.6 }}>
        {name.slice(0, 1).toUpperCase()}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({ center: { alignItems: "center", justifyContent: "center" } });
