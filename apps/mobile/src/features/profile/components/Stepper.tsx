import { View } from 'react-native';
import { useStyles } from '../../../theme';
import { IconButton, Text, icons } from '../../../ui';

interface StepperProps {
  label: string;
  valueText: string;
  /** 0..1 position shown on the track. */
  progress: number;
  onDecrement: () => void;
  onIncrement: () => void;
  decrementLabel: string;
  incrementLabel: string;
  canDecrement?: boolean;
  canIncrement?: boolean;
}

/** Slider-like control: label, big value, +/- buttons and a progress track. */
export function Stepper({
  label,
  valueText,
  progress,
  onDecrement,
  onIncrement,
  decrementLabel,
  incrementLabel,
  canDecrement = true,
  canIncrement = true,
}: StepperProps) {
  const styles = useStyles((t) => ({
    wrap: { gap: t.spacing.sm },
    row: {
      flexDirection: 'row',
      alignItems: 'center',
      justifyContent: 'space-between',
      gap: t.spacing.md,
    },
    value: { flex: 1, alignItems: 'center' },
    track: { height: 6, borderRadius: 3, backgroundColor: t.colors.surfaceAlt, overflow: 'hidden' },
    fill: { height: 6, borderRadius: 3, backgroundColor: t.colors.primary },
  }));
  return (
    <View
      style={styles.wrap}
      accessible
      accessibilityRole="adjustable"
      accessibilityLabel={`${label}: ${valueText}`}
      accessibilityActions={[
        { name: 'increment', label: incrementLabel },
        { name: 'decrement', label: decrementLabel },
      ]}
      onAccessibilityAction={(e) => {
        if (e.nativeEvent.actionName === 'increment' && canIncrement) onIncrement();
        if (e.nativeEvent.actionName === 'decrement' && canDecrement) onDecrement();
      }}
    >
      <Text variant="label">{label}</Text>
      <View style={styles.row}>
        <IconButton
          glyph={icons.minus}
          accessibilityLabel={decrementLabel}
          onPress={onDecrement}
          disabled={!canDecrement}
          tone="outline"
        />
        <View style={styles.value}>
          <Text variant="heading" color="primary">
            {valueText}
          </Text>
        </View>
        <IconButton
          glyph={icons.plus}
          accessibilityLabel={incrementLabel}
          onPress={onIncrement}
          disabled={!canIncrement}
          tone="outline"
        />
      </View>
      <View
        style={styles.track}
        accessibilityElementsHidden
        importantForAccessibility="no-hide-descendants"
      >
        <View
          style={[
            styles.fill,
            { width: `${Math.round(Math.min(1, Math.max(0, progress)) * 100)}%` },
          ]}
        />
      </View>
    </View>
  );
}
