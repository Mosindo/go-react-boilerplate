jest.mock("expo-secure-store", () => ({
  getItemAsync: jest.fn(async () => null),
  setItemAsync: jest.fn(async () => undefined),
  deleteItemAsync: jest.fn(async () => undefined)
}));

jest.mock("expo-linear-gradient", () => ({ LinearGradient: "LinearGradient" }));

// Icons load fonts asynchronously, which causes act() noise; a stub is enough for logic tests.
jest.mock("@expo/vector-icons/Ionicons", () => {
  const { Text } = jest.requireActual("react-native");
  return { __esModule: true, default: Text };
});
