import { useRef, useState } from 'react';
import type { ReactNode } from 'react';
import { View } from 'react-native';
import { errorMessage } from '../../api';
import { fr } from '../../i18n';
import type { TabScreenProps } from '../../navigation/types';
import { useStyles } from '../../theme';
import { EmptyState, ErrorState, IconButton, Screen, Skeleton, Text, icons } from '../../ui';
import { useDeck } from './DeckContext';
import { SwipeDeck } from './SwipeDeck';
import type { DeckHandle } from './SwipeDeck';

export function DiscoverScreen({ navigation }: TabScreenProps<'Discover'>) {
  const { state, loadError, act, retry } = useDeck();
  const deckHandle = useRef<DeckHandle>(null);
  const [busy, setBusy] = useState(false);
  const styles = useStyles((t) => ({
    header: {
      flexDirection: 'row',
      alignItems: 'center',
      gap: t.spacing.sm,
      paddingVertical: t.spacing.md,
    },
    stage: { flex: 1, paddingBottom: t.spacing.md },
    actions: {
      flexDirection: 'row',
      justifyContent: 'center',
      gap: t.spacing.xxl,
      paddingVertical: t.spacing.lg,
    },
    skeleton: { flex: 1, paddingBottom: t.spacing.lg },
  }));

  const hasCards = state.queue.length > 0;

  let body: ReactNode;
  if (hasCards) {
    body = (
      <>
        <View style={styles.stage}>
          <SwipeDeck
            profiles={state.queue}
            handle={deckHandle}
            onBusyChange={setBusy}
            onSwipe={act}
            onOpen={(profile) =>
              navigation.navigate('ProfileDetail', {
                userId: profile.userId,
                profile,
                source: 'discover',
              })
            }
          />
        </View>
        <View style={styles.actions}>
          <IconButton
            glyph={icons.close}
            accessibilityLabel={fr.discover.pass}
            size={64}
            tone="outline"
            color="danger"
            disabled={busy}
            onPress={() => deckHandle.current?.swipe('pass')}
            testID="discover-pass"
          />
          <IconButton
            glyph={icons.heart}
            accessibilityLabel={fr.discover.like}
            size={64}
            tone="primary"
            disabled={busy}
            onPress={() => deckHandle.current?.swipe('like')}
            testID="discover-like"
          />
        </View>
      </>
    );
  } else if (loadError !== null) {
    body = <ErrorState message={errorMessage(loadError)} onRetry={retry} />;
  } else if (state.exhausted) {
    body = (
      <EmptyState
        glyph={icons.spark}
        title={fr.discover.emptyTitle}
        message={fr.discover.emptyMessage}
        actionLabel={fr.discover.emptyAction}
        onAction={() => navigation.navigate('Preferences')}
      />
    );
  } else {
    body = (
      <View
        style={styles.skeleton}
        accessibilityRole="progressbar"
        accessibilityLabel={fr.common.loading}
      >
        <Skeleton height="100%" radius={28} style={{ flex: 1 }} />
      </View>
    );
  }

  return (
    <Screen edges={['top']}>
      <View style={styles.header}>
        <Text color="primary" style={{ fontSize: 24 }} accessible={false}>
          {icons.spark}
        </Text>
        <Text variant="heading" color="primary">
          {fr.brand.name}
        </Text>
      </View>
      {body}
    </Screen>
  );
}
