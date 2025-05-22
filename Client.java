import java.io.*;
import java.net.Socket;
import java.net.UnknownHostException;
import java.util.Scanner;
import java.util.List;
import java.util.ArrayList;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.Arrays;


public class Client {

    private static final int SERVER_PORT = 12345;
    private static final String DOWNLOAD_DIRECTORY = "files";
    private static final String UPLOAD_DIRECTORY = "upload_files";

    public static void main(String[] args) {
        Scanner scanner = new Scanner(System.in);
        System.out.print("Enter server address: ");
        String serverAddress = scanner.nextLine();

        createDirectoryIfNotExists(DOWNLOAD_DIRECTORY);
        createDirectoryIfNotExists(UPLOAD_DIRECTORY);
        populateUploadDirectory(); // For demonstration purposes in client 'upload_files' dir

        try (Socket socket = new Socket(serverAddress, SERVER_PORT);
             DataInputStream dataInputStream = new DataInputStream(socket.getInputStream());
             DataOutputStream dataOutputStream = new DataOutputStream(socket.getOutputStream())) {

            System.out.println("Connected to server: " + serverAddress);

            sendFilesToServer(dataOutputStream); // Send files to server first
            receiveFilesFromServer(dataInputStream); // Then receive files from server


        } catch (UnknownHostException e) {
            System.err.println("Unknown host: " + serverAddress);
        } catch (IOException e) {
            System.err.println("Could not connect to server or IO error: " + e.getMessage());
        } finally {
            scanner.close();
        }
    }

    private static void createDirectoryIfNotExists(String dirName) {
        File directory = new File(dirName);
        if (!directory.exists()) {
            directory.mkdirs();
        }
    }

     // Create some dummy files for testing if the directory is empty
    private static void populateUploadDirectory() {
        File filesDir = new File(UPLOAD_DIRECTORY);
        if (filesDir.list().length == 0) {
            for (int i = 1; i <= 3; i++) {
                File dummyFile = new File(filesDir, "client_file" + i + ".txt"); // Differentiate client files
                if (!dummyFile.exists()) {
                    try (FileWriter writer = new FileWriter(dummyFile)) {
                        writer.write("This is client file number " + i + ".");
                    } catch (IOException e) {
                        e.printStackTrace();
                    }
                }
            }
        }
    }

    private static void sendFilesToServer(DataOutputStream dataOutputStream) throws IOException {
        List<File> filesToSend = getFileList(UPLOAD_DIRECTORY);
        dataOutputStream.writeInt(filesToSend.size()); // Send number of files
        System.out.println("Sending " + filesToSend.size() + " files to server.");

        for (File fileToSend : filesToSend) {
            dataOutputStream.writeUTF(fileToSend.getName()); // Send file name
            dataOutputStream.writeLong(fileToSend.length()); // Send file size

            try (FileInputStream fileInputStream = new FileInputStream(fileToSend)) {
                byte[] buffer = new byte[4096];
                int bytesRead;
                while ((bytesRead = fileInputStream.read(buffer)) != -1) {
                    dataOutputStream.write(buffer, 0, bytesRead);
                }
            }
            dataOutputStream.flush();
            System.out.println("File sent to server: " + fileToSend.getName());
        }
        System.out.println("All files sent to server.");
    }

    private static void receiveFilesFromServer(DataInputStream dataInputStream) throws IOException {
        // Receive file list
        int fileCount = dataInputStream.readInt();
        System.out.println("Number of files to download from server: " + fileCount);
        String[] fileNames = new String[fileCount];
        for (int i = 0; i < fileCount; i++) {
            fileNames[i] = dataInputStream.readUTF();
            System.out.println("File to download from server: " + fileNames[i]);
        }

        // Download files
        for (int i = 0; i < fileCount; i++) {
            String originalFileName = dataInputStream.readUTF(); // Get original filename from server
            long fileSize = dataInputStream.readLong();

            System.out.println("Downloading file from server: " + originalFileName + ", size: " + fileSize + " bytes");

            byte[] fileContent = new byte[(int) fileSize]; // Read all content to calculate hash
            dataInputStream.readFully(fileContent); // Ensure all bytes are read

            String sha256Hash = calculateSHA256(fileContent);
            String fileExtension = getFileExtension(originalFileName);
            String hashedFileName = sha256Hash + "." + fileExtension;
            File downloadedFile = new File(DOWNLOAD_DIRECTORY, hashedFileName);


            try (FileOutputStream fileOutputStream = new FileOutputStream(downloadedFile)) {
                fileOutputStream.write(fileContent); // Write from the content buffer
            }
            System.out.println("File downloaded and saved as: " + hashedFileName);
        }

        String downloadStatus = dataInputStream.readUTF();
        if ("DOWNLOAD_COMPLETE".equals(downloadStatus)) {
            System.out.println("Download completed successfully.");
        } else {
            System.out.println("Download status: " + downloadStatus);
        }
    }


    private static List<File> getFileList(String directoryPath) {
        List<File> files = new ArrayList<>();
        File directory = new File(directoryPath);
        File[] fileArray = directory.listFiles();
        if (fileArray != null) {
            for (File file : fileArray) {
                if (file.isFile()) {
                    files.add(file);
                }
            }
        }
        return files;
    }


    private static String calculateSHA256(byte[] fileContent) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hashBytes = digest.digest(fileContent);
            StringBuilder hexString = new StringBuilder();
            for (byte hashByte : hashBytes) {
                String hex = Integer.toHexString(0xff & hashByte);
                if (hex.length() == 1) hexString.append('0');
                hexString.append(hex);
            }
            return hexString.toString();
        } catch (NoSuchAlgorithmException e) {
            e.printStackTrace(); // Handle exception properly in real app
            return null;
        }
    }

    private static String getFileExtension(String filename) {
        if (filename == null || filename.lastIndexOf('.') == -1) {
            return "dat"; // default extension if no extension found
        }
        return filename.substring(filename.lastIndexOf('.') + 1);
    }
}