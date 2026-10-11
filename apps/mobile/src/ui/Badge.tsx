import { View } from 'react-native';
import { formatUnread } from '../lib/format';
import { useStyles } from '../theme';
import { Text } from './Text';

export function Badge({ count }: { count: number }) {
  const styles = useStyles((t) => ({
    badge: {
      minWidth: 22,
      height: 22,
      borderRadius: 11,
      paddingHorizontal: 6,
      backgroundColor: t.colors.primary,
      alignItems: 'center',
      justifyContent: 'center',
    },
  }));
  if (count <= 0) return null;
  return (
    <View style={styles.badge}>
      <Text
        variant="caption"
        color="textOnPrimary"
        weight="700"
        style={{ fontSize: 12, lineHeight: 16 }}
      >
        {formatUnread(count)}
      </Text>
    </View>
  );
}
