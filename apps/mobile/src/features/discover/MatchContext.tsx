import { createContext, useCallback, useContext, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { Animated, Modal, View } from 'react-native';
import { useEffect, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import type { NavigationContainerRefWithCurrent } from '@react-navigation/native';
import { queryKeys } from '../../api';
import type { Conversation, OwnProfile, PublicProfile } from '../../api';
import { fr } from '../../i18n';
import { USE_NATIVE_DRIVER } from '../../lib/motion';
import type { RootStackParamList } from '../../navigation/types';
import { MAX_CONTENT_WIDTH, useStyles } from '../../theme';
import { Avatar, Button, Text, icons } from '../../ui';

interface MatchContextValue {
  /** Opens the "It's a match" modal. `profile` is used for the photo when we have it. */
  showMatch: (conversation: Conversation, profile?: PublicProfile) => void;
}

const MatchContext = createContext<MatchContextValue | null>(null);

interface MatchState {
  conversation: Conversation;
  photoUrl: string | null;
}

export function MatchProvider({
  children,
  navigationRef,
}: {
  children: ReactNode;
  navigationRef: NavigationContainerRefWithCurrent<RootStackParamList>;
}) {
  const [match, setMatch] = useState<MatchState | null>(null);
  const qc = useQueryClient();

  const showMatch = useCallback<MatchContextValue['showMatch']>((conversation, profile) => {
    setMatch({
      conversation,
      photoUrl: profile?.photos[0]?.url ?? conversation.user.photoUrl,
    });
  }, []);

  const value = useMemo(() => ({ showMatch }), [showMatch]);
  const own = match ? qc.getQueryData<OwnProfile>(queryKeys.ownProfile) : undefined;

  return (
    <MatchContext.Provider value={value}>
      {children}
      <MatchModal
        match={match}
        ownPhotoUrl={own?.photos[0]?.url ?? null}
        ownName={own?.firstName ?? ''}
        onClose={() => setMatch(null)}
        onChat={() => {
          if (!match) return;
          const { conversation } = match;
          setMatch(null);
          if (navigationRef.isReady()) navigationRef.navigate('Chat', { conversation });
        }}
      />
    </MatchContext.Provider>
  );
}

export function useMatch(): MatchContextValue {
  const ctx = useContext(MatchContext);
  if (!ctx) throw new Error('useMatch must be used inside <MatchProvider>');
  return ctx;
}

function MatchModal({
  match,
  ownPhotoUrl,
  ownName,
  onClose,
  onChat,
}: {
  match: MatchState | null;
  ownPhotoUrl: string | null;
  ownName: string;
  onClose: () => void;
  onChat: () => void;
}) {
  const pop = useRef(new Animated.Value(0)).current;
  const styles = useStyles((t) => ({
    backdrop: {
      flex: 1,
      backgroundColor: t.colors.scrim,
      alignItems: 'center',
      justifyContent: 'center',
      padding: t.spacing.xl,
    },
    card: {
      width: '100%',
      maxWidth: MAX_CONTENT_WIDTH - 48,
      backgroundColor: t.colors.surface,
      borderRadius: t.radius.xl,
      padding: t.spacing.xl,
      alignItems: 'center',
      gap: t.spacing.lg,
    },
    avatars: { flexDirection: 'row', alignItems: 'center' },
    heart: {
      width: 44,
      height: 44,
      borderRadius: 22,
      backgroundColor: t.colors.primary,
      alignItems: 'center',
      justifyContent: 'center',
      marginHorizontal: -10,
      zIndex: 2,
    },
    actions: { alignSelf: 'stretch', gap: t.spacing.sm },
  }));

  useEffect(() => {
    if (match) {
      pop.setValue(0);
      Animated.spring(pop, {
        toValue: 1,
        friction: 6,
        tension: 80,
        useNativeDriver: USE_NATIVE_DRIVER,
      }).start();
    }
  }, [match, pop]);

  if (!match) return null;
  const name = match.conversation.user.firstName;
  return (
    <Modal transparent visible animationType="fade" onRequestClose={onClose} statusBarTranslucent>
      <View style={styles.backdrop}>
        <Animated.View
          accessibilityViewIsModal
          accessibilityLiveRegion="polite"
          style={[
            styles.card,
            {
              transform: [
                { scale: pop.interpolate({ inputRange: [0, 1], outputRange: [0.85, 1] }) },
              ],
              opacity: pop,
            },
          ]}
        >
          <Text variant="title" color="primary" align="center">
            {fr.match.title}
          </Text>
          <View style={styles.avatars}>
            <Avatar url={ownPhotoUrl} name={ownName || fr.match.you} size={96} />
            <View style={styles.heart} accessible={false}>
              <Text color="textOnPrimary" style={{ fontSize: 22 }}>
                {icons.heart}
              </Text>
            </View>
            <Avatar url={match.photoUrl} name={name} size={96} />
          </View>
          <Text align="center" color="textMuted">
            {fr.match.subtitle(name)}
          </Text>
          <View style={styles.actions}>
            <Button label={fr.match.sendMessage} onPress={onChat} testID="match-open-chat" />
            <Button
              label={fr.match.keepSwiping}
              variant="ghost"
              onPress={onClose}
              testID="match-close"
            />
          </View>
        </Animated.View>
      </View>
    </Modal>
  );
}
