import React from "react";
import { StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { absoluteUrl } from "../config";
import { colors } from "../theme/tokens";
import { AppText } from "./AppText";

type Props = { url?: string | null; name: string; size?: number; ring?: boolean };

export function Avatar({ url, name, size = 52, ring }: Props) {
  const style = { width: size, height: size, borderRadius: size / 2 };
  return (
    <View style={[styles.frame, style, ring && styles.ring]} accessibilityLabel={`Photo de ${name}`}>
      {url ? (
        <Image source={{ uri: absoluteUrl(url) }} style={style} contentFit="cover" transition={150} cachePolicy="memory-disk" />
      ) : (
        <AppText variant="heading" color={colors.textMuted}>
          {name.slice(0, 1).toUpperCase()}
        </AppText>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  frame: { backgroundColor: colors.surfaceRaised, alignItems: "center", justifyContent: "center", overflow: "hidden" },
  ring: { borderWidth: 2, borderColor: colors.primary }
});
