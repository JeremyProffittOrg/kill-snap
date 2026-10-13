"""Exercise the deployed ZIP writer using only isolated generated test data."""
import io
import json
import time
import uuid
import zipfile

import boto3


def main():
    session = boto3.Session(region_name="us-east-2")
    if session.client("sts").get_caller_identity()["Account"] != "759775734231":
        raise RuntimeError("Wrong account")
    s3, ddb, functions = session.client("s3"), session.resource("dynamodb"), session.client("lambda")
    token = "cost-smoke-" + uuid.uuid4().hex
    prefix = "project-zips/" + token + "/"
    bucket = "kill-snap"
    projects, images = ddb.Table("kill-snap-Projects"), ddb.Table("kill-snap-ImageMetadata")
    payloads = {"source.png": b"synthetic ZIP smoke source", "source.dng": b"synthetic RAW smoke source"}
    try:
        for name, data in payloads.items():
            s3.put_object(Bucket=bucket, Key=prefix + name, Body=data)
        projects.put_item(Item={"ProjectID": token, "Name": "Cost Smoke", "S3Prefix": token, "ImageCount": 1}, ConditionExpression="attribute_not_exists(ProjectID)")
        images.put_item(Item={"ImageGUID": token, "ProjectID": token, "OriginalFile": prefix + "source.png", "FileSize": len(payloads["source.png"])}, ConditionExpression="attribute_not_exists(ImageGUID)")
        for attempt in range(10):
            found = images.query(IndexName="ProjectIndex", KeyConditionExpression=boto3.dynamodb.conditions.Key("ProjectID").eq(token))
            if len(found["Items"]) == 1:
                break
            time.sleep(1)
        else:
            raise RuntimeError("Fixture index did not converge")
        result = functions.invoke(FunctionName="ProjectZipGenerator", Payload=json.dumps({"projectId": token}).encode())
        result["Payload"].read()
        if result.get("FunctionError"):
            raise RuntimeError("Deployed ZIP invocation failed")
        project = projects.get_item(Key={"ProjectID": token}, ConsistentRead=True)["Item"]
        archives = project.get("ZipFiles", [])
        if len(archives) != 1 or archives[0]["Status"] != "complete" or not archives[0]["Key"].startswith(prefix):
            raise RuntimeError("ZIP did not complete under its lifecycle prefix")
        obj = s3.get_object(Bucket=bucket, Key=archives[0]["Key"])
        with zipfile.ZipFile(io.BytesIO(obj["Body"].read())) as archive:
            if archive.read("source.png") != payloads["source.png"] or archive.read("source.dng") != payloads["source.dng"]:
                raise RuntimeError("Generated ZIP contents differ")
            if "source.xmp" not in archive.namelist():
                raise RuntimeError("RAW XMP sidecar missing")
        print("PASS: deployed ZIP uses expiring prefix and preserves synthetic source/RAW/XMP")
    finally:
        # Only this invocation's random fixture IDs and prefix can be removed.
        images.delete_item(Key={"ImageGUID": token})
        projects.delete_item(Key={"ProjectID": token})
        for page in s3.get_paginator("list_object_versions").paginate(Bucket=bucket, Prefix=prefix):
            versions = page.get("Versions", []) + page.get("DeleteMarkers", [])
            for obj in versions:
                if not obj["Key"].startswith(prefix):
                    raise RuntimeError("Unsafe fixture cleanup key")
                s3.delete_object(Bucket=bucket, Key=obj["Key"], VersionId=obj["VersionId"])
        print("PASS: isolated smoke fixture removed")


if __name__ == "__main__":
    main()
