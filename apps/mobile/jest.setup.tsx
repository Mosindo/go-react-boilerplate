jest.mock('expo-secure-store', () => {
  const store = new Map<string, string>();
  return {
    setItemAsync: async (k: string, v: string) => void store.set(k, v),
    getItemAsync: async (k: string) => store.get(k) ?? null,
    deleteItemAsync: async (k: string) => void store.delete(k),
  };
});

jest.mock('@expo/vector-icons', () => {
  const { Text } = jest.requireActual('react-native');
  const Ionicons = ({ name }: { name: string }) => <Text>{name}</Text>;
  Ionicons.glyphMap = {};
  return { Ionicons };
});
