import { View } from 'react-native';
import { errorMessage } from '../../../api';
import { fr } from '../../../i18n';
import { MAX_INTERESTS } from '../../../lib/validation';
import { useStyles } from '../../../theme';
import { Chip, ErrorState, Skeleton, Text } from '../../../ui';
import { useInterests } from '../hooks';

interface InterestPickerProps {
  selectedIds: number[];
  onChange: (ids: number[]) => void;
}

export function InterestPicker({ selectedIds, onChange }: InterestPickerProps) {
  const interests = useInterests();
  const styles = useStyles((t) => ({
    wrap: { gap: t.spacing.md },
    chips: { flexDirection: 'row', flexWrap: 'wrap', gap: t.spacing.sm },
    head: { flexDirection: 'row', justifyContent: 'space-between', alignItems: 'center' },
  }));
  const full = selectedIds.length >= MAX_INTERESTS;

  const toggle = (id: number) => {
    if (selectedIds.includes(id)) onChange(selectedIds.filter((x) => x !== id));
    else if (!full) onChange([...selectedIds, id]);
  };

  if (interests.isError) {
    return (
      <ErrorState
        message={errorMessage(interests.error)}
        onRetry={() => void interests.refetch()}
      />
    );
  }

  return (
    <View style={styles.wrap}>
      <View style={styles.head}>
        <Text variant="label">{fr.profile.interests}</Text>
        <Text
          variant="caption"
          color={full ? 'primary' : 'textMuted'}
          accessibilityLiveRegion="polite"
        >
          {fr.profile.interestsCount(selectedIds.length, MAX_INTERESTS)}
        </Text>
      </View>
      <View style={styles.chips}>
        {interests.isPending
          ? Array.from({ length: 10 }, (_, i) => (
              <Skeleton key={i} width={72 + (i % 4) * 16} height={44} radius={22} />
            ))
          : interests.data.map((interest) => {
              const selected = selectedIds.includes(interest.id);
              return (
                <Chip
                  key={interest.id}
                  label={interest.label}
                  selected={selected}
                  disabled={!selected && full}
                  onPress={() => toggle(interest.id)}
                />
              );
            })}
      </View>
    </View>
  );
}
