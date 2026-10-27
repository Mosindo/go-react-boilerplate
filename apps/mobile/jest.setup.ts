jest.mock("expo-secure-store", () => {
  const store = new Map<string, string>();
  return {
    setItemAsync: async (k: string, v: string) => void store.set(k, v),
    getItemAsync: async (k: string) => store.get(k) ?? null,
    deleteItemAsync: async (k: string) => void store.delete(k)
  };
});
jest.mock("expo-constants", () => ({ __esModule: true, default: { expoConfig: {} } }));
jest.mock("expo-location", () => ({}));
jest.mock("expo-image-picker", () => ({}));
