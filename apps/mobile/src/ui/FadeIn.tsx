import React, { useEffect, useRef } from "react";
import { Animated, type ViewStyle } from "react-native";

type Props = { children: React.ReactNode; delay?: number; style?: ViewStyle };

/** Fade + small rise on mount. */
export function FadeIn({ children, delay = 0, style }: Props) {
  const progress = useRef(new Animated.Value(0)).current;
  useEffect(() => {
    Animated.timing(progress, { toValue: 1, duration: 280, delay, useNativeDriver: true }).start();
  }, [progress, delay]);
  return (
    <Animated.View
      style={[
        style,
        {
          opacity: progress,
          transform: [
            { translateY: progress.interpolate({ inputRange: [0, 1], outputRange: [8, 0] }) }
          ]
        }
      ]}
    >
      {children}
    </Animated.View>
  );
}
