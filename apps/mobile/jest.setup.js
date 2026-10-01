jest.mock("expo-secure-store", () => ({
  setItemAsync: jest.fn(async () => undefined),
  getItemAsync: jest.fn(async () => null),
  deleteItemAsync: jest.fn(async () => undefined)
}));

process.env.EXPO_PUBLIC_API_URL = "http://api.test";

// Icon fonts load asynchronously and would trigger act() warnings; icons are decorative here.
jest.mock("@expo/vector-icons", () => ({ Ionicons: () => null }));
