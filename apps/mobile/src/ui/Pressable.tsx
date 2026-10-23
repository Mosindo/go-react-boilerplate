import React, { useRef } from "react";
import {
  Animated,
  Pressable as RNPressable,
  type PressableProps,
  type ViewStyle
} from "react-native";

type Props = Omit<PressableProps, "style"> & { style?: ViewStyle | ViewStyle[] };

/** Pressable with a subtle press-scale animation. */
export function ScalePressable({ style, onPressIn, onPressOut, disabled, ...rest }: Props) {
  const scale = useRef(new Animated.Value(1)).current;
  const animate = (to: number) =>
    Animated.spring(scale, {
      toValue: to,
      useNativeDriver: true,
      speed: 40,
      bounciness: 4
    }).start();
  return (
    <Animated.View style={[{ transform: [{ scale }] }, style]}>
      <RNPressable
        {...rest}
        disabled={disabled}
        onPressIn={(event) => {
          animate(0.97);
          onPressIn?.(event);
        }}
        onPressOut={(event) => {
          animate(1);
          onPressOut?.(event);
        }}
        style={{ minHeight: 44 }}
      />
    </Animated.View>
  );
}
