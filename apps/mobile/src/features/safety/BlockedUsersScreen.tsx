import { FlatList, RefreshControl, View } from 'react-native';
import { errorMessage } from '../../api';
import type { BlockedUser } from '../../api';
import { fr } from '../../i18n';
import { useStyles, useTheme } from '../../theme';
import {
  Avatar,
  Button,
  Card,
  EmptyState,
  ErrorState,
  Loading,
  Screen,
  Text,
  icons,
} from '../../ui';
import { useBlocks, useUnblockUser } from './hooks';

export function BlockedUsersScreen() {
  const { colors } = useTheme();
  const blocks = useBlocks();
  const unblock = useUnblockUser();
  const styles = useStyles((t) => ({
    list: { paddingVertical: t.spacing.lg, gap: t.spacing.md, flexGrow: 1 },
    row: { flexDirection: 'row', alignItems: 'center', gap: t.spacing.md },
    name: { flex: 1 },
  }));

  const renderItem = ({ item }: { item: BlockedUser }) => (
    <Card>
      <View style={styles.row}>
        <Avatar url={null} name={item.firstName} size={44} />
        <View style={styles.name}>
          <Text weight="600">{item.firstName}</Text>
        </View>
        <Button
          label={fr.safety.unblock}
          variant="ghost"
          size="sm"
          fullWidth={false}
          accessibilityLabel={fr.safety.unblockLabel(item.firstName)}
          loading={unblock.isPending && unblock.variables === item.userId}
          onPress={() => unblock.mutate(item.userId)}
        />
      </View>
    </Card>
  );

  if (blocks.isPending) return <Loading />;
  if (blocks.isError) {
    return (
      <Screen edges={['bottom']}>
        <ErrorState message={errorMessage(blocks.error)} onRetry={() => void blocks.refetch()} />
      </Screen>
    );
  }
  return (
    <Screen edges={['bottom']}>
      <FlatList
        data={blocks.data}
        keyExtractor={(item) => item.userId}
        renderItem={renderItem}
        contentContainerStyle={styles.list}
        refreshControl={
          <RefreshControl
            refreshing={blocks.isRefetching}
            onRefresh={() => void blocks.refetch()}
            tintColor={colors.primary}
            colors={[colors.primary]}
          />
        }
        ListHeaderComponent={<Text color="textMuted">{fr.safety.blockedIntro}</Text>}
        ListEmptyComponent={
          <EmptyState
            glyph={icons.lock}
            title={fr.safety.blockedEmptyTitle}
            message={fr.safety.blockedEmptyMessage}
          />
        }
      />
    </Screen>
  );
}
