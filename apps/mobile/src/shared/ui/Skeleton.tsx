import React, { useEffect } from "react";
import { Animated, type StyleProp, type ViewStyle, useAnimatedValue } from "react-native";
import { radius, useTheme } from "../../theme";

type Props = {
  width?: ViewStyle["width"];
  height?: ViewStyle["height"];
  round?: boolean;
  style?: StyleProp<ViewStyle>;
};

export function Skeleton({ width = "100%", height = 16, round = false, style }: Props) {
  const { colors } = useTheme();
  const opacity = useAnimatedValue(0.45);

  useEffect(() => {
    const loop = Animated.loop(
      Animated.sequence([
        Animated.timing(opacity, { toValue: 1, duration: 800, useNativeDriver: true }),
        Animated.timing(opacity, { toValue: 0.45, duration: 800, useNativeDriver: true })
      ])
    );
    loop.start();
    return () => loop.stop();
  }, [opacity]);

  return (
    <Animated.View
      accessibilityElementsHidden
      importantForAccessibility="no-hide-descendants"
      style={[
        { width, height, opacity, backgroundColor: colors.surfaceAlt, borderRadius: round ? radius.pill : radius.sm },
        style
      ]}
    />
  );
}
