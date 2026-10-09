import React from 'react';
import { StyleSheet, View } from 'react-native';
import { fr } from '../../i18n/fr';
import { spacing } from '../../theme';
import { useInterestCatalog } from '../../query/hooks';
import { Chip, ErrorState, LoadingView, Text } from '../ui';

export const MAX_INTERESTS = 10;

export function InterestPicker({
  selected,
  onChange,
}: {
  selected: string[];
  onChange: (slugs: string[]) => void;
}) {
  const { data, isPending, isError, refetch } = useInterestCatalog();
  if (isPending) return <LoadingView />;
  if (isError) return <ErrorState title={fr.interests.loadFailed} onRetry={() => void refetch()} />;
  const full = selected.length >= MAX_INTERESTS;
  return (
    <View>
      <Text variant="label" tone="muted" style={styles.count} accessibilityLiveRegion="polite">
        {fr.interests.count(selected.length)}
      </Text>
      <View style={styles.wrap}>
        {data.map((i) => {
          const on = selected.includes(i.slug);
          return (
            <Chip
              key={i.slug}
              testID={`interest-${i.slug}`}
              label={i.label}
              selected={on}
              disabled={!on && full}
              onPress={() =>
                onChange(on ? selected.filter((s) => s !== i.slug) : [...selected, i.slug])
              }
            />
          );
        })}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  count: { marginBottom: spacing.sm },
  wrap: { flexDirection: 'row', flexWrap: 'wrap' },
});
